# SIMD Merge Sort

This project implements a vectorized merge sort algorithm using AVX-512 instructions. It provides both a standard Go implementation and an AVX-512 optimized version for comparing performance.

## Features

- Vectorized merge sort using AVX-512 instructions
- Standard Go implementation for comparison
- Benchmarking tools to measure performance
- Support for key-value pairs with integer keys and string values

## Requirements

- Go 1.21 or later
- CPU with AVX-512 support
- Linux operating system (for AVX-512 assembly)

## Building

1. Clone the repository:
```bash
git clone https://github.com/yourusername/simdmerge.git
cd simdmerge
```

2. Install dependencies:
```bash
go mod download
```

3. Generate the AVX-512 assembly code:
```bash
go generate
```

4. Build the project:
```bash
go build
```

## Usage

Run the benchmark program:
```bash
./simdmerge
```

The program will:
1. Generate test data of various sizes
2. Run both standard and vectorized merge sort implementations
3. Compare performance and verify results
4. Print benchmark results and system information

## Implementation Details

- The vectorized implementation uses AVX-512 instructions to process 16 32-bit integers at once
- The merge operation is performed using vectorized comparisons and masked operations
- The standard implementation uses a traditional merge sort algorithm
- Both implementations handle key-value pairs with integer keys and string values

## Performance

The vectorized implementation typically shows significant speedup over the standard implementation, especially for larger datasets. The exact speedup depends on:
- Input size
- CPU architecture
- Memory access patterns
- Data distribution

## License

MIT License 