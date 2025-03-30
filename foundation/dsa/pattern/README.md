# Recursive Patterns

Common recursive patterns along with specific examples for each.

## 1. Tree Traversal Patterns
```
    A
   / \
  B   C
 / \
D   E
```
### Preorder Traversal
- **Definition**: Visit the root node first, then recursively visit the left subtree, followed by the right subtree.
- **Example**: For a binary tree:
The preorder traversal would be: **A, B, D, E, C**.

### Inorder Traversal
- **Definition**: Recursively visit the left subtree, then the root node, and finally the right subtree.
- **Example**: For the same binary tree:
The inorder traversal would be: **D, B, E, A, C**.

### Postorder Traversal
- **Definition**: Recursively visit the left subtree, then the right subtree, and finally the root node.
- **Example**: For the same binary tree:
The postorder traversal would be: **D, E, B, C, A**.

## 2. Backtracking
- **Definition**: A method to find all solutions by exploring possible options and backtracking upon reaching an invalid state.
- **Example**: The N-Queens problem, where you need to place N queens on an N×N chessboard such that no two queens threaten each other.

## 3. Divide and Conquer
- **Definition**: Breaks a problem into smaller subproblems, solves each subproblem independently, and combines the results.
- **Example**: Merge Sort algorithm:
    - Split the array into two halves.
    - Recursively sort both halves.
    - Merge the sorted halves.

## 4. Dynamic Programming
- **Definition**: A method to solve complex problems by breaking them down into simpler subproblems, storing the results of subproblems to avoid redundant work.
- **Example**: Fibonacci sequence calculation:
```go
func fib(n int) int {
    if n <= 1 {
        return n
    }
    return fib(n-1) + fib(n-2)
}
```
With memoization, store previously calculated results.
## 5. Depth-First Search (DFS)
- **Definition**: A traversal method for graphs or trees where you explore as far as possible along each branch before backtracking.
- **Example**: Performing DFS on a graph:

```
A -- B
|    |
C -- D
```
Starting from A, the DFS might visit: A, B, D, C.
## 6. Generate Combinations and Permutations
- **Definition**: Using recursion to generate all possible combinations or permutations of a set of elements.
- **Example**: Generating all permutations of the set {1, 2, 3}:
Result: {123, 132, 213, 231, 312, 321}.
## 7. Recursive Descent Parsing
- **Definition**: A top-down method of parsing where each rule of a grammar is represented by a recursive function.
- **Example**: Parsing a simple arithmetic expression grammar:
Rules could include expressions, terms, and factors, each represented by a recursive function.
## 8. Memoization
- **Definition**: A technique where you store the results of expensive function calls and reuse them when the same inputs occur again.
- **Example**: In the Fibonacci function:
```go
var memo = make(map[int]int)

func fib(n int) int {
    if n <= 1 {
        return n
    }
    if result, found := memo[n]; found {
        return result
    }
    memo[n] = fib(n-1) + fib(n-2)
    return memo[n]
}
```
## Summary
These patterns are essential in various algorithms and problem-solving techniques. They provide a structured way to approach complex recursive problems efficiently. If you have any specific questions or need further examples, feel free to ask!