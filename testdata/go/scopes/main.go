package main

import "fmt"

// Global variables for testing global scope
var globalString = "I am a global string"
var globalInt = 42

type Person struct {
	Name string
	Age  int
}

func main() {
	// Local variables in main
	localVar := "main local"
	number := 100

	// Use the local variables
	fmt.Println(localVar, number)

	// Call a function with arguments
	result := greet("Alice", 30)
	fmt.Println(result)

	// Create a struct
	person := Person{Name: "Bob", Age: 25}
	processPerson(person)

	// Test with slice and map
	numbers := []int{1, 2, 3, 4, 5}
	data := map[string]int{"one": 1, "two": 2, "three": 3}

	processCollection(numbers, data)
}

func greet(name string, age int) string {
	// Function with arguments and local variables
	greeting := fmt.Sprintf("Hello %s, you are %d years old", name, age)
	prefix := "Greeting: "

	fullGreeting := prefix + greeting

	// Breakpoint location for testing scopes
	return fullGreeting // Set breakpoint here (line 42)
}

func processPerson(p Person) {
	// Local variables
	description := fmt.Sprintf("%s is %d years old", p.Name, p.Age)
	isAdult := p.Age >= 18

	// Breakpoint location for testing scopes with struct parameter
	fmt.Println(description, "Adult:", isAdult) // Set breakpoint here (line 54)
}

func processCollection(nums []int, dict map[string]int) {
	// Local variables
	sum := 0
	for _, n := range nums {
		sum += n
	}

	count := len(dict)

	// Breakpoint location for testing scopes with collection parameters
	fmt.Printf("Sum: %d, Map entries: %d\n", sum, count) // Set breakpoint here (line 67)
}
