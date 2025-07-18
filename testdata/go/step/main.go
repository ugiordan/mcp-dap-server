package main

import "fmt"

func main() {
	// Line 6: Initialize first variable
	x := 10

	// Line 9: Initialize second variable
	y := 20

	// Line 12: Perform calculation
	sum := x + y

	// Line 15: Create a string
	message := fmt.Sprintf("Sum is: %d", sum)

	// Line 18: Print result
	fmt.Println(message)

	// Line 21: Modify variables
	x = x * 2
	y = y + 5

	// Line 25: Another calculation
	product := x * y

	// Line 28: Final print
	fmt.Printf("Product is: %d\n", product)
}
