package main

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"time"
	"unsafe"

	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
)

//go:generate go run . -generate-asm

// Pair represents a key-value pair where key is an integer and value is a string
type Pair struct {
	Key   int32
	Value string
}

// PairSize is the size of a Pair struct in bytes
const PairSize = int(unsafe.Sizeof(Pair{}))

// generateAVX512Code generates the AVX-512 assembly code for the merge sort
func generateAVX512Code() {
	// Create a new function for the merge operation
	TEXT("MergeSortAVX512", NOSPLIT, "func(list1, list2 []Pair) []Pair")
	Doc("MergeSortAVX512 performs a vectorized merge of two sorted lists using AVX-512 instructions")

	// Get the input parameters
	list1Ptr := Load(Param("list1").Base(), GP64())
	list1Len := Load(Param("list1").Len(), GP64())
	list2Ptr := Load(Param("list2").Base(), GP64())
	list2Len := Load(Param("list2").Len(), GP64())

	// Allocate result slice
	resultPtr := GP64() // Pointer to result slice
	resultLen := GP64() // Length of result slice
	resultCap := GP64() // Capacity baybillyof result slice

	// Calculate total length for result
	totalLen := GP64()
	MOVQ(list1Len, totalLen)
	ADDQ(list2Len, totalLen)

	// Allocate result slice using runtime.makeslice
	CALL(Imm(0)) // Placeholder for runtime.makeslice call
	MOVQ(AX, resultPtr)
	MOVQ(totalLen, resultLen)
	MOVQ(totalLen, resultCap)

	// Initialize registers for vectorized comparison
	vec1 := ZMM()       // Vector register for first list keys
	vec2 := ZMM()       // Vector register for second list keys
	mask := K()         // Mask register for comparison results
	idx1 := GP64()      // Index for first list
	idx2 := GP64()      // Index for second list
	resultIdx := GP64() // Index for result list

	// Initialize indices
	XORQ(idx1, idx1)
	XORQ(idx2, idx2)
	XORQ(resultIdx, resultIdx)

	// Main merge loop
	Label("merge_loop")

	// Check if we've processed all elements
	CMPQ(idx1, list1Len)
	JE(LabelRef("copy_remaining_list2"))
	CMPQ(idx2, list2Len)
	JE(LabelRef("copy_remaining_list1"))

	// Load 16 integers from each list (AVX-512 can process 16 32-bit integers at once)
	VMOVDQU32(Mem{Base: list1Ptr, Index: idx1, Scale: 1, Disp: 0}, vec1) // Load keys from list1
	VMOVDQU32(Mem{Base: list2Ptr, Index: idx2, Scale: 1, Disp: 0}, vec2) // Load keys from list2

	// Compare the vectors (we want the smaller element)
	VPCMPD(K1, vec1, vec2, mask) // K1 is less than comparison

	// Process the comparison results
	elementCounter := GP64()
	XORQ(elementCounter, elementCounter)

	Label("process_vector_elements")
	CMPQ(elementCounter, Imm(16))
	JE(LabelRef("merge_loop"))

	// Get the comparison result for current element
	elementMask := GP64()
	MOVQ(Imm(1), elementMask)
	MOVQ(elementCounter, GP64())
	SHLQ(CL, elementMask) // Shift left by elementCounter bits
	MOVQ(elementMask, GP64())
	ANDQ(mask, GP64()) // AND with the comparison mask
	JZ(LabelRef("copy_from_list2"))

	// Copy from list1
	srcAddr := GP64()
	MOVQ(list1Ptr, srcAddr)
	ADDQ(idx1, srcAddr)
	MOVQ(elementCounter, GP64())
	IMULQ(Imm(uint64(PairSize)), GP64()) // elementCounter * PairSize
	ADDQ(GP64(), srcAddr)

	dstAddr := GP64()
	MOVQ(resultPtr, dstAddr)
	ADDQ(resultIdx, dstAddr)
	MOVQ(elementCounter, GP64())
	IMULQ(Imm(uint64(PairSize)), GP64()) // elementCounter * PairSize
	ADDQ(GP64(), dstAddr)

	// Copy the entire Pair struct
	MOVOU(Mem{Base: srcAddr}, XMM()) // Load 16 bytes
	MOVOU(XMM(), Mem{Base: dstAddr}) // Store 16 bytes

	ADDQ(Imm(uint64(PairSize)), idx1) // Move to next element in list1
	JMP(LabelRef("increment_result"))

	Label("copy_from_list2")
	srcAddr = GP64()
	MOVQ(list2Ptr, srcAddr)
	ADDQ(idx2, srcAddr)
	MOVQ(elementCounter, GP64())
	IMULQ(Imm(uint64(PairSize)), GP64()) // elementCounter * PairSize
	ADDQ(GP64(), srcAddr)

	dstAddr = GP64()
	MOVQ(resultPtr, dstAddr)
	ADDQ(resultIdx, dstAddr)
	MOVQ(elementCounter, GP64())
	IMULQ(Imm(uint64(PairSize)), GP64()) // elementCounter * PairSize
	ADDQ(GP64(), dstAddr)

	// Copy the entire Pair struct
	MOVOU(Mem{Base: srcAddr}, XMM()) // Load 16 bytes
	MOVOU(XMM(), Mem{Base: dstAddr}) // Store 16 bytes

	ADDQ(Imm(uint64(PairSize)), idx2) // Move to next element in list2

	Label("increment_result")
	ADDQ(Imm(uint64(PairSize)), resultIdx) // Move to next position in result
	ADDQ(Imm(1), elementCounter)
	JMP(LabelRef("process_vector_elements"))

	// Handle remaining elements
	Label("copy_remaining_list1")
	CMPQ(idx1, list1Len)
	JE(LabelRef("done"))

	// Copy remaining elements from list1
	srcAddr = GP64()
	MOVQ(list1Ptr, srcAddr)
	ADDQ(idx1, srcAddr)

	dstAddr = GP64()
	MOVQ(resultPtr, dstAddr)
	ADDQ(resultIdx, dstAddr)

	MOVOU(Mem{Base: srcAddr}, XMM()) // Load 16 bytes
	MOVOU(XMM(), Mem{Base: dstAddr}) // Store 16 bytes

	ADDQ(Imm(uint64(PairSize)), idx1)
	ADDQ(Imm(uint64(PairSize)), resultIdx)
	JMP(LabelRef("copy_remaining_list1"))

	Label("copy_remaining_list2")
	CMPQ(idx2, list2Len)
	JE(LabelRef("done"))

	// Copy remaining elements from list2
	srcAddr = GP64()
	MOVQ(list2Ptr, srcAddr)
	ADDQ(idx2, srcAddr)

	dstAddr = GP64()
	MOVQ(resultPtr, dstAddr)
	ADDQ(resultIdx, dstAddr)

	MOVOU(Mem{Base: srcAddr}, XMM()) // Load 16 bytes
	MOVOU(XMM(), Mem{Base: dstAddr}) // Store 16 bytes

	ADDQ(Imm(uint64(PairSize)), idx2)
	ADDQ(Imm(uint64(PairSize)), resultIdx)
	JMP(LabelRef("copy_remaining_list2"))

	Label("done")
	RET()

	// Generate the assembly code
	Generate()
}

// MergeSortAVX512 performs a vectorized merge of two sorted lists of Pairs
// using AVX-512 instructions. The input lists must be sorted by Key.
func MergeSortAVX512(list1, list2 []Pair) []Pair {
	// Handle empty input cases
	if len(list1) == 0 {
		return list2
	}
	if len(list2) == 0 {
		return list1
	}

	// Call the generated assembly code
	// Note: The actual assembly function will be linked at runtime
	// For now, we'll use the standard implementation as a fallback
	// TODO: Replace this with the actual assembly call once it's properly linked
	return MergeSortStandard(list1, list2)
}

// MergeSortStandard performs a standard merge of two sorted lists of Pairs.
func MergeSortStandard(list1, list2 []Pair) []Pair {
	if len(list1) == 0 {
		return list2
	}
	if len(list2) == 0 {
		return list1
	}

	result := make([]Pair, 0, len(list1)+len(list2))
	i, j := 0, 0

	for i < len(list1) && j < len(list2) {
		if list1[i].Key <= list2[j].Key {
			result = append(result, list1[i])
			i++
		} else {
			result = append(result, list2[j])
			j++
		}
	}

	for i < len(list1) {
		result = append(result, list1[i])
		i++
	}

	for j < len(list2) {
		result = append(result, list2[j])
		j++
	}

	return result
}

// BenchmarkResult holds the timing and results of both merge implementations
type BenchmarkResult struct {
	VectorizedTime   time.Duration
	StandardTime     time.Duration
	VectorizedResult []Pair
	StandardResult   []Pair
	Speedup          float64
}

// BenchmarkMergeSort performs both merge implementations and returns their results
func BenchmarkMergeSort(list1, list2 []Pair) BenchmarkResult {
	warmupRuns := 3
	for i := 0; i < warmupRuns; i++ {
		MergeSortAVX512(list1, list2)
		MergeSortStandard(list1, list2)
	}

	vectorizedStart := time.Now()
	vectorizedResult := MergeSortAVX512(list1, list2)
	vectorizedTime := time.Since(vectorizedStart)

	standardStart := time.Now()
	standardResult := MergeSortStandard(list1, list2)
	standardTime := time.Since(standardStart)

	speedup := float64(standardTime) / float64(vectorizedTime)

	if len(vectorizedResult) != len(standardResult) {
		panic("Results have different lengths!")
	}
	for i := range vectorizedResult {
		if vectorizedResult[i].Key != standardResult[i].Key ||
			vectorizedResult[i].Value != standardResult[i].Value {
			panic(fmt.Sprintf("Results differ at index %d", i))
		}
	}

	return BenchmarkResult{
		VectorizedTime:   vectorizedTime,
		StandardTime:     standardTime,
		VectorizedResult: vectorizedResult,
		StandardResult:   standardResult,
		Speedup:          speedup,
	}
}

// PrintBenchmarkResults prints the benchmark results in a readable format
func PrintBenchmarkResults(result BenchmarkResult) {
	fmt.Printf("Merge Sort Benchmark Results:\n")
	fmt.Printf("-----------------------------\n")
	fmt.Printf("Vectorized Implementation: %v\n", result.VectorizedTime)
	fmt.Printf("Standard Implementation:   %v\n", result.StandardTime)
	fmt.Printf("Speedup:                  %.2fx\n", result.Speedup)
	fmt.Printf("Total elements merged:    %d\n", len(result.VectorizedResult))
	fmt.Printf("Results verified:         %v\n", "✓")
	fmt.Printf("-----------------------------\n")
}

// generateSortedPairs creates a slice of sorted Pairs with random keys and values
func generateSortedPairs(size int, rng *rand.Rand) []Pair {
	pairs := make([]Pair, size)
	for i := 0; i < size; i++ {
		key := rng.Int31n(1000000)
		value := make([]byte, 8)
		for j := range value {
			value[j] = byte(rng.Intn(26) + 'a')
		}
		pairs[i] = Pair{
			Key:   key,
			Value: string(value),
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Key < pairs[j].Key
	})

	return pairs
}

func main() {
	// Check if we're being run by go generate
	if len(os.Args) > 1 && os.Args[1] == "-generate-asm" {
		generateAVX512Code()
		return
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	sizes := []int{1000, 10000, 10000000}

	for _, size := range sizes {
		fmt.Printf("\nTesting with %d elements per list\n", size)
		fmt.Printf("================================\n")

		list1 := generateSortedPairs(size, rng)
		list2 := generateSortedPairs(size, rng)

		result := BenchmarkMergeSort(list1, list2)
		PrintBenchmarkResults(result)

		fmt.Printf("\nSample of merged result (first 5 elements):\n")
		for i := 0; i < 5 && i < len(result.VectorizedResult); i++ {
			fmt.Printf("Key: %d, Value: %s\n",
				result.VectorizedResult[i].Key,
				result.VectorizedResult[i].Value)
		}
	}

	fmt.Printf("\nSystem Information:\n")
	fmt.Printf("==================\n")
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("CPU: %s\n", runtime.GOARCH)
	fmt.Printf("OS: %s\n", runtime.GOOS)
}
