# SOLID Principles

SOLID is an acronym for five design principles intended to make software designs more understandable, flexible, and maintainable. These principles were introduced by Robert C. Martin, also known as Uncle Bob.

## S - Single Responsibility Principle (SRP)

A class should have only one reason to change, meaning that a class should have only one job or responsibility.

**Example**:
```go
package solid

// Before SRP
type User struct {
    Name  string
    Email string
}

func (u *User) Save() {
    // Save user to database
}

func (u *User) SendEmail() {
    // Send email to user
}

// After SRP
type User struct {
    Name  string
    Email string
}

type UserRepository struct{}

func (r *UserRepository) Save(u *User) {
    // Save user to database
}

type EmailService struct{}

func (e *EmailService) SendEmail(u *User) {
    // Send email to user
}
```

## O - Open/Closed Principle (OCP)
Software entities (classes, modules, functions, etc.) should be open for extension but closed for modification.

Example:
```go
package solid

// Before OCP
type Rectangle struct {
    Width, Height float64
}

type Circle struct {
    Radius float64
}

func Area(shape interface{}) float64 {
    switch s := shape.(type) {
    case Rectangle:
        return s.Width * s.Height
    case Circle:
        return 3.14 * s.Radius * s.Radius
    }
    return 0
}

// After OCP
type Shape interface {
    Area() float64
}

type Rectangle struct {
    Width, Height float64
}

func (r Rectangle) Area() float64 {
    return r.Width * r.Height
}

type Circle struct {
    Radius float64
}

func (c Circle) Area() float64 {
    return 3.14 * c.Radius * c.Radius
}
```
## L - Liskov Substitution Principle (LSP)
Objects of a superclass should be replaceable with objects of a subclass without affecting the correctness of the program.

Example:
```go
package solid

type Bird interface {
    Fly() string
}

type Sparrow struct{}

func (s Sparrow) Fly() string {
    return "Sparrow flying"
}

type Ostrich struct{}

func (o Ostrich) Fly() string {
    return "Ostrich can't fly"
}
```
## I - Interface Segregation Principle (ISP)
Clients should not be forced to depend on interfaces they do not use. Instead of one fat interface, many small interfaces are preferred based on groups of methods with specific behaviors.

Example:
```go
package solid

// Before ISP
type Worker interface {
    Work()
    Eat()
}

type Human struct{}

func (h Human) Work() {
    // Human working
}

func (h Human) Eat() {
    // Human eating
}

type Robot struct{}

func (r Robot) Work() {
    // Robot working
}

func (r Robot) Eat() {
    // Robot doesn't eat
}

// After ISP
type Worker interface {
    Work()
}

type Eater interface {
    Eat()
}

type Human struct{}

func (h Human) Work() {
    // Human working
}

func (h Human) Eat() {
    // Human eating
}

type Robot struct{}

func (r Robot) Work() {
    // Robot working
}
```
## D - Dependency Inversion Principle (DIP)
High-level modules should not depend on low-level modules. Both should depend on abstractions. Abstractions should not depend on details. Details should depend on abstractions.

Example:
```go
package solid

// Before DIP
type LightBulb struct{}

func (l LightBulb) TurnOn() {
    // Turn on the light bulb
}

func (l LightBulb) TurnOff() {
    // Turn off the light bulb
}

type Switch struct {
    bulb LightBulb
}

func (s Switch) Operate() {
    s.bulb.TurnOn()
}

// After DIP
type Switchable interface {
    TurnOn()
    TurnOff()
}

type LightBulb struct{}

func (l LightBulb) TurnOn() {
    // Turn on the light bulb
}

func (l LightBulb) TurnOff() {
    // Turn off the light bulb
}

type Switch struct {
    device Switchable
}

func (s Switch) Operate() {
    s.device.TurnOn()
}
```

## Conclusion
The SOLID principles are fundamental guidelines for designing robust, maintainable, and scalable software. By adhering to these principles, developers can create systems that are easier to understand, extend, and maintain over time.