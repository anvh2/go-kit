# Prefix Sum Pattern

The Prefix Sum pattern is a common technique used in data structures and algorithms to preprocess an array of numbers to enable efficient range sum queries. This pattern is particularly useful for solving problems that involve frequent queries for the sum of elements in a subarray.

## Key Concepts

- **Prefix Sum Array**: An auxiliary array where each element at index `i` contains the sum of the elements from the start of the original array up to index `i`.
- **Range Sum Query**: A query that asks for the sum of elements between two indices in the array.

## How It Works

1. **Compute the Prefix Sum Array**:
   - Create an array `prefix` of the same length as the original array.
   - Initialize `prefix[0]` to `arr[0]`.
   - For each subsequent element, compute `prefix[i]` as `prefix[i-1] + arr[i]`.

2. **Answer Range Sum Queries**:
   - For a query asking for the sum of elements between indices `l` and `r`, use the formula:
     - `sum(l, r) = prefix[r] - prefix[l-1]` if `l > 0`
     - `sum(l, r) = prefix[r]` if `l == 0`

## Example

Given an array `arr = [1, 2, 3, 4, 5]`, the prefix sum array is computed as follows:

- `prefix[0] = 1`
- `prefix[1] = 1 + 2 = 3`
- `prefix[2] = 3 + 3 = 6`
- `prefix[3] = 6 + 4 = 10`
- `prefix[4] = 10 + 5 = 15`

The prefix sum array is `[1, 3, 6, 10, 15]`.

To find the sum of elements from index `1` to `3` (i.e., `2 + 3 + 4`):

- `sum(1, 3) = prefix[3] - prefix[0] = 10 - 1 = 9`

## Code Example

Here is a simple implementation in Go:

```go
package prefixsum

// ComputePrefixSum computes the prefix sum array for the given array.
func ComputePrefixSum(arr []int) []int {
    prefix := make([]int, len(arr))
    prefix[0] = arr[0]
    for i := 1; i < len(arr); i++ {
        prefix[i] = prefix[i-1] + arr[i]
    }
    return prefix
}

// RangeSum returns the sum of elements between indices l and r (inclusive).
func RangeSum(prefix []int, l, r int) int {
    if l == 0 {
        return prefix[r]
    }
    return prefix[r] - prefix[l-1]
}