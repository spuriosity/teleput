# Teleput: A Go Learning Guide for Python Engineers

> A deep-dive textbook covering Go fundamentals, terminal UI architecture, event-driven design, and the Elm Architecture — all taught through teleput, a terminal file browser for [put.io](https://put.io). Written for senior Python engineers who want to understand Go at a level where they can reason about, debug, and extend real Go code.

---

## Table of Contents

### Part I: Go Language Foundations
1. [The 30-Second Mental Model Shift](#1-the-30-second-mental-model-shift)
2. [Compilation, Binaries, and the Runtime](#2-compilation-binaries-and-the-runtime)
3. [Package System and Project Layout](#3-package-system-and-project-layout)
4. [The Type System: Static but Surprisingly Flexible](#4-the-type-system-static-but-surprisingly-flexible)
5. [Structs, Methods, and the Death of Classes](#5-structs-methods-and-the-death-of-classes)
6. [Interfaces: Duck Typing with Teeth](#6-interfaces-duck-typing-with-teeth)
7. [Zero Values: The Most Important Thing Python Doesn't Have](#7-zero-values-the-most-important-thing-python-doesnt-have)
8. [Error Handling: The Philosophy of Explicit Failure](#8-error-handling-the-philosophy-of-explicit-failure)
9. [Value vs Pointer Semantics: Who Owns the Data?](#9-value-vs-pointer-semantics-who-owns-the-data)
10. [Slices, Maps, and Collection Semantics](#10-slices-maps-and-collection-semantics)
11. [Closures and First-Class Functions](#11-closures-and-first-class-functions)
12. [Defer, Panic, and Resource Management](#12-defer-panic-and-resource-management)
13. [Concurrency: Goroutines, Channels, and Select](#13-concurrency-goroutines-channels-and-select)
14. [Context: Cancellation, Deadlines, and Request Scope](#14-context-cancellation-deadlines-and-request-scope)
15. [Visibility, Constants, and the Package-Level Namespace](#15-visibility-constants-and-the-package-level-namespace)
16. [Dependency Management: go.mod and the Module System](#16-dependency-management-gomod-and-the-module-system)
17. [Memory Model: Stack, Heap, and Escape Analysis](#17-memory-model-stack-heap-and-escape-analysis)

### Part II: Terminal UI Foundations
18. [How Terminals Actually Work](#18-how-terminals-actually-work)
19. [ANSI Escape Sequences and Styled Output](#19-ansi-escape-sequences-and-styled-output)
20. [The Event Loop: From Raw Input to Application State](#20-the-event-loop-from-raw-input-to-application-state)
21. [The Elm Architecture: A General Theory of UI](#21-the-elm-architecture-a-general-theory-of-ui)
22. [Unidirectional Data Flow vs Traditional Event Handling](#22-unidirectional-data-flow-vs-traditional-event-handling)

### Part III: Teleput Deep Dive
23. [Application Lifecycle: From `main()` to Pixel](#23-application-lifecycle-from-main-to-pixel)
24. [The Configuration Cascade](#24-the-configuration-cascade)
25. [OAuth Out-of-Band: Polling as Concurrency](#25-oauth-out-of-band-polling-as-concurrency)
26. [The Model Layer: State as a Value](#26-the-model-layer-state-as-a-value)
27. [The Browser Component: Sub-Models and Delegation](#27-the-browser-component-sub-models-and-delegation)
28. [Commands as Deferred Side Effects](#28-commands-as-deferred-side-effects)
29. [Type Switches: Pattern Matching for Messages](#29-type-switches-pattern-matching-for-messages)
30. [Styling with Lip Gloss: Immutable Builders and Theming](#30-styling-with-lip-gloss-immutable-builders-and-theming)
31. [The OAuth2 Token Source Pattern](#31-the-oauth2-token-source-pattern)
32. [Sorting, Formatting, and Standard Library Patterns](#32-sorting-formatting-and-standard-library-patterns)
33. [Architectural Patterns and the Composition Root](#33-architectural-patterns-and-the-composition-root)

### Appendices
- [A: Python-Go Quick Reference](#appendix-a-python-go-quick-reference)
- [B: Go Concepts Not in Teleput](#appendix-b-go-concepts-not-in-teleput)
- [C: Exercises](#appendix-c-exercises)

---

# Part I: Go Language Foundations

## 1. The 30-Second Mental Model Shift

If Python is a dynamic Swiss Army knife — flexible, forgiving, let-me-do-anything — then Go is a precision torque wrench. It does fewer things, but every one of those things clicks at exactly the right tightness.

The core mental shifts:

- **No classes.** Structs + methods + interfaces replace the entire OOP hierarchy. Think of it like Python's `dataclass` + standalone functions that happen to receive `self`, but without inheritance.
- **No exceptions.** Errors are values. Every function that can fail returns `(result, error)`. You *check* errors; you don't *catch* them. It's like if Python forced you to write `result, err = do_thing()` and the language wouldn't let you ignore `err`.
- **No generics needed (mostly).** The standard library is designed around interfaces so thoroughly that you rarely miss generics (though Go has had them since 1.18).
- **Concurrency is a first-class citizen.** Where Python has `asyncio` bolted on top, Go has goroutines and channels baked into the language syntax itself.
- **Compilation to a single binary.** No virtualenv, no `requirements.txt` at deploy time. `go build` gives you one file. Ship it.

The analogy I keep coming back to: Python gives you a blank canvas and every colour ever invented. Go gives you a drafting table and a #2 pencil — and it turns out you can build remarkably sophisticated things with constraint.

The teleput codebase is ~250 lines across 5 files. Every one of those lines teaches something.

---

## 2. Compilation, Binaries, and the Runtime

### How Go code becomes an executable

Python is interpreted: `python main.py` reads source, compiles to bytecode, executes on the CPython VM. Go is **ahead-of-time compiled**: `go build` produces a native machine code binary.

```
Python:  source.py  →  bytecode (.pyc)  →  CPython VM interprets
Go:      source.go  →  machine code      →  OS executes directly
```

When you run `go build` in teleput's directory, you get a single `teleput` binary (~10-15MB) that contains:
- All of teleput's code, compiled to native instructions
- All dependencies (Bubble Tea, Lip Gloss, put.io client, oauth2)
- The **Go runtime** (goroutine scheduler, garbage collector, stack management)

There's no equivalent of `pip install` at deploy time. You copy the binary to a server and run it. That's it.

### The Go runtime is not a VM

Python's CPython is a virtual machine that interprets bytecode. Go's runtime is fundamentally different — it's a set of support routines **linked into your binary** that provide:

- **Goroutine scheduler**: Multiplexes thousands of goroutines onto a handful of OS threads (M:N scheduling). This is like having a built-in, highly optimised `asyncio` event loop, but it works with regular blocking code — no `async/await` syntax needed.
- **Garbage collector**: Concurrent, tri-color mark-and-sweep. Unlike Python's reference counting + cycle collector, Go's GC runs concurrently with your program, targeting sub-millisecond pause times. For a TUI like teleput, this means no perceptible GC pauses.
- **Stack management**: Goroutine stacks start at ~2-4KB and grow dynamically (segmented stacks, now contiguous since Go 1.4). A Python thread stack is typically 8MB. This 2000x difference is why you can run millions of goroutines but only thousands of threads.
- **Network poller**: Integrates with the OS's I/O multiplexer (epoll/kqueue) so that goroutines can block on network I/O without blocking an OS thread.

### Cross-compilation

```bash
GOOS=linux GOARCH=amd64 go build -o teleput-linux
GOOS=darwin GOARCH=arm64 go build -o teleput-mac
GOOS=windows GOARCH=amd64 go build -o teleput.exe
```

Two environment variables, and you get a binary for any platform. No Docker, no CI matrix, no "works on my machine." This is something Python simply cannot do.

### The `go run` shortcut

During development, `go run main.go` compiles and runs in one step — similar to `python main.py` in feel, but the compilation step happens transparently. The Go compiler is fast enough (~100ms for small projects) that this feels instant.

---

## 3. Package System and Project Layout

### How teleput is organised

```
teleput/
  main.go            # package main — the entry point
  go.mod             # dependency manifest (like pyproject.toml)
  go.sum             # dependency lockfile (like uv.lock)
  auth/
    oauth.go         # package auth
  config/
    config.go        # package config
  ui/
    model.go         # package ui
    browser.go       # package ui
    theme.go         # package ui
```

### The directory-is-a-package rule

In Python, each `.py` file is a module. In Go, **each directory is a package**, and all `.go` files in that directory share the same package namespace. This is a fundamental difference.

```python
# Python: each file is its own namespace
# auth/oauth.py
class OAuthClient: ...

# You import the file
from auth.oauth import OAuthClient
```

```go
// Go: all files in auth/ share "package auth"
// auth/oauth.go
package auth

func Authenticate(ctx context.Context) (string, error) { ... }

// You import the package, not the file
import "github.com/jack/teleput/auth"
auth.Authenticate(ctx)
```

Three things to notice:

1. **The file name doesn't matter for imports.** You import `auth`, not `auth/oauth`. If you added `auth/helpers.go` with `package auth`, its exported symbols would be available through the same `auth.XXX` path. All files in the directory merge into one namespace. This is why teleput's `ui` package is split across three files (`model.go`, `browser.go`, `theme.go`) — they all contribute to one `ui` namespace.

2. **`package main` is special.** It tells the Go compiler "this is an executable, not a library." The `main()` function in `package main` is the entry point — analogous to `if __name__ == "__main__"` but enforced by the compiler.

3. **Import paths are URLs.** The module path in `go.mod` (`github.com/jack/teleput`) is both the import path *and* where `go get` will look for the source code. There's no separate registry like PyPI — the VCS host *is* the registry.

### The import block

```go
// main.go:3-13
import (
    "context"       // stdlib
    "flag"          // stdlib
    "fmt"           // stdlib
    "os"            // stdlib

    tea "github.com/charmbracelet/bubbletea"  // aliased import

    "github.com/jack/teleput/auth"            // local package
    "github.com/jack/teleput/config"          // local package
    "github.com/jack/teleput/ui"              // local package
)
```

The `tea "github.com/charmbracelet/bubbletea"` syntax is an **import alias** — equivalent to Python's `import bubbletea as tea`. This is idiomatic in the Charm ecosystem; everyone calls Bubble Tea `tea`.

Go enforces **import grouping** by convention: stdlib, then third-party, then local packages, separated by blank lines. The `goimports` tool does this automatically (like `isort` for Python).

One critical rule: **Go will not compile if you have an unused import.** This seems annoying at first, but it means every import in teleput is actually used. No dead imports. Ever. (During development, you can use `_ "pkg"` as a blank import to suppress the error temporarily.)

> **Reference:** `main.go:1-14` (package declaration + imports), `go.mod:1` (module path)

---

## 4. The Type System: Static but Surprisingly Flexible

### Foundational types

Go's built-in types map to Python like this:

| Go | Python | Key difference |
|---|---|---|
| `int`, `int8`, `int16`, `int32`, `int64` | `int` | Go integers have fixed sizes; Python's are arbitrary precision |
| `float32`, `float64` | `float` | Go has two; Python has one (64-bit) |
| `string` | `str` | Both are immutable; Go strings are byte sequences (not necessarily UTF-8) |
| `bool` | `bool` | Identical semantics |
| `byte` (alias for `uint8`) | `bytes[i]` | A single byte |
| `rune` (alias for `int32`) | `str[i]` (a char) | A single Unicode code point |

The `int64` choice for file IDs in teleput is deliberate:

```go
// ui/browser.go:19-20
parentID int64
parents  []int64
```

In Go, `int` is platform-dependent (32 or 64 bits). The put.io API returns file IDs that could exceed 32-bit range, so `int64` is explicit. In Python, you never think about integer overflow because `int` has arbitrary precision. In Go, you always think about this.

### Type declarations create new types

```go
// ui/model.go:71
type errMsg struct{ err error }
```

This declares a **new type** `errMsg`. It's not an alias — it's a distinct type. You can't pass an `errMsg` where a different struct with the same fields is expected. This matters because Go uses type identity for dispatch in type switches.

### Multiple return values

Every Python developer who sees this pattern for the first time does a double-take:

```go
// config/config.go:24-41
func Load() (*Config, error) {
    dir, err := Dir()
    if err != nil {
        return nil, err
    }
    data, err := os.ReadFile(filepath.Join(dir, "config.json"))
    // ...
}
```

Go functions return **multiple values natively** — it's not a tuple being unpacked, it's actual multi-value return at the language level. The compiler tracks each value independently. The `(result, error)` pair is the dominant pattern.

### `:=` vs `var` — short variable declaration

```go
token := *tokenFlag          // type inferred from right-hand side
var cmd tea.Cmd              // explicit type, zero value
```

`:=` is **short variable declaration** — it declares *and* initialises in one step, with the type inferred. It's like Python's bare `x = 5`, but the variable is freshly declared, not reassigned. You can only use `:=` inside functions.

`var` is the long form: `var x Type = value`. You use it when you want to specify the type explicitly or when you want the zero value without an initialiser.

A subtle gotcha: `:=` can **shadow** outer variables in inner scopes:

```go
x := 1
if condition {
    x := 2  // this is a NEW x, shadowing the outer one
    // outer x is still 1
}
// x is still 1 here
```

This is different from Python, where assignment in an inner scope modifies the enclosing scope (or you use `nonlocal`). Go's scoping is lexical and strict.

---

## 5. Structs, Methods, and the Death of Classes

### Structs are data containers

A struct in Go is closest to a Python `dataclass` or `NamedTuple` — a typed bag of fields with no inheritance.

```go
// config/config.go:11-13
type Config struct {
    OAuthToken string `json:"oauth_token"`
}
```

```go
// ui/model.go:12-19
type Model struct {
    client   *putio.Client
    width    int
    height   int
    browser  browserModel
    err      error
    quitting bool
}
```

Python equivalent:

```python
@dataclass
class Model:
    client: putio.Client
    width: int = 0
    height: int = 0
    browser: BrowserModel = field(default_factory=BrowserModel)
    err: Exception | None = None
    quitting: bool = False
```

But there's a crucial difference: **structs have no constructors.** There's no `__init__`. You either use a struct literal or write a factory function by convention.

```go
// ui/model.go:21-29
func NewModel(token string) Model {
    ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
    httpClient := oauth2.NewClient(context.Background(), ts)
    client := putio.NewClient(httpClient)

    return Model{
        client:  client,
        browser: newBrowserModel(client),
    }
}
```

The `NewXxx` convention is Go's replacement for `__init__`. It's just a function — there's no magic, no special method resolution order, no metaclasses. The capital `N` makes it exported (public). Compare `NewModel` (exported, called from `main.go`) with `newBrowserModel` (unexported, only used within `package ui`).

### Methods are functions with a receiver

In Python, methods live inside the class body and receive `self` implicitly. In Go, methods are **declared outside the struct** and receive the struct explicitly:

```go
// ui/model.go:32-34
func (m Model) Init() tea.Cmd {
    return m.browser.loadDir(0)
}
```

The `(m Model)` part is the **receiver**. It's literally just `self` with a type annotation, declared before the function name. By convention, Go uses a short abbreviation of the type (`m` for Model, `f` for File) rather than `self` or `this`.

The profound difference: **methods in Go don't require a class body.** You can define methods on any type you own, in any file within the same package. Teleput splits its `ui` package across three files — `model.go`, `browser.go`, `theme.go` — and they all define things in the same namespace. This is like being able to add methods to a class from different files.

### No inheritance, only composition

Go has no `class Dog(Animal):` inheritance. Instead, you compose structs:

```go
// Not in teleput, but illustrative:
type Animal struct {
    Name string
}

type Dog struct {
    Animal      // embedded — Dog "has an" Animal
    Breed string
}
```

With embedding, `Dog` gets `Animal`'s fields and methods promoted to its own surface. `dog.Name` works without `dog.Animal.Name`. This is **structural composition**, not inheritance — there's no virtual dispatch, no method resolution order, no diamond problem.

Teleput doesn't use embedding because its structs are simple enough. But the pattern is everywhere in Go: `http.Client` embeds a `Transport`, `bufio.Scanner` wraps an `io.Reader`, etc.

> **Reference:** `ui/model.go:12-29` (struct + factory), `config/config.go:11-13` (simple struct)

---

## 6. Interfaces: Duck Typing with Teeth

This is the single most important concept in Go for a Python developer. It's where Go's philosophy diverges most beautifully from Python's.

### Implicit interface satisfaction

In Python, you implement an interface by inheriting from an ABC or Protocol:

```python
from typing import Protocol

class Renderable(Protocol):
    def render(self) -> str: ...

class MyWidget:  # implicitly satisfies Protocol (structural subtyping)
    def render(self) -> str:
        return "hello"
```

In Go, **you never declare that you implement an interface.** If your type has the right methods, it satisfies the interface. Period. No `implements` keyword, no inheritance, no registration.

### How this works in practice

The most important interface in teleput is `tea.Model`:

```go
// Defined by Bubble Tea (you never see this in teleput's code):
type Model interface {
    Init() Cmd
    Update(Msg) (Model, Cmd)
    View() string
}
```

Teleput's `Model` struct satisfies this interface because it has all three methods:

```go
func (m Model) Init() tea.Cmd { ... }                        // ui/model.go:32
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { ... }  // ui/model.go:36
func (m Model) View() string { ... }                         // ui/model.go:60
```

The compiler checks structural compatibility at the call site — when teleput passes `Model` to `tea.NewProgram()`, the compiler verifies that `Model` has all three methods with the right signatures. If you remove one, you get a compile error pointing at `tea.NewProgram(m, ...)` saying "Model does not implement tea.Model (missing method Update)."

The analogy: **Python's duck typing says "if it walks like a duck, use it like a duck" at runtime and hopes for the best. Go's interfaces say the same thing, but the compiler proves it's a duck before the program ever runs.**

### Small interfaces, big power

Go interfaces tend to be tiny. The most pervasive ones:

```go
type error interface {
    Error() string
}

type Stringer interface {  // fmt package
    String() string
}

type Reader interface {  // io package
    Read(p []byte) (n int, err error)
}

type Writer interface {  // io package
    Write(p []byte) (n int, err error)
}
```

The `error` interface is one method. `io.Reader` is one method. This design philosophy — **interfaces should have one or two methods** — is radically different from Java's 20-method interfaces or Python ABCs with dozens of abstract methods.

The result: small interfaces compose. `io.ReadCloser` is just:

```go
type ReadCloser interface {
    Reader
    Closer
}
```

Two interfaces composed into one. No multiple inheritance, no MRO headaches.

### The empty interface: `interface{}` and `any`

```go
type Msg interface{}  // Bubble Tea's message type
```

The empty interface is satisfied by every type — it's Go's `object` or `Any`. Since Go 1.18, you can write `any` instead of `interface{}` (they're aliases). Bubble Tea uses this so that any type can be a message: `tea.WindowSizeMsg`, `tea.KeyMsg`, `errMsg`, `filesLoadedMsg` — they're all concrete types that satisfy `Msg` trivially.

The downside: you lose type safety. The `Update` function receives a `tea.Msg` and must use a type switch (Section 29) to recover the concrete type. This is Go's version of Python's `if isinstance(msg, SomeType)`.

> **Reference:** `ui/model.go:32-69` (interface satisfaction), `ui/model.go:71` (errMsg as a Msg)

---

## 7. Zero Values: The Most Important Thing Python Doesn't Have

In Python, uninitialised variables don't exist — accessing one is a `NameError`. In Go, every variable has a **zero value** determined by its type:

| Type | Zero value | Python equivalent |
|---|---|---|
| `int`, `int64` | `0` | N/A (no equivalent) |
| `string` | `""` | N/A |
| `bool` | `false` | N/A |
| `*T` (pointer) | `nil` | `None` |
| slice (`[]T`) | `nil` (empty, usable) | `None` (NOT `[]`, must initialise) |
| struct | All fields zero-valued | N/A |
| `error` | `nil` | N/A |

### A Go design idiom: make the zero value useful

This is a proverb from the Go community. Teleput's `browserModel` demonstrates it:

```go
// ui/browser.go:31-34
func newBrowserModel(client *putio.Client) browserModel {
    return browserModel{
        client: client,
    }
}
```

Only `client` is explicitly set. All other fields get zero values:

- `files []putio.File` → `nil` (empty slice, safe to iterate and append to)
- `cursor int` → `0` (starts at first item — correct)
- `parentID int64` → `0` (put.io's root directory is ID 0 — **this is by design**)
- `parents []int64` → `nil` (empty navigation stack — correct)
- `loading bool` → `false` (not loading yet — correct)

The root directory being ID 0 is not a coincidence — the code is designed so that the zero value of `parentID` correctly points to the root. This is the Go idiom in action: design your types so the zero value is meaningful.

Another example: `strings.Builder`'s zero value is a ready-to-use empty builder. You don't need `strings.NewBuilder()` — just `var b strings.Builder` and start writing to it (see `ui/browser.go:100`).

### `nil` is not Python's `None`

`nil` in Go is the zero value for pointers, slices, maps, channels, interfaces, and functions. But unlike `None`:

- `nil` has no single type — it takes the type of whatever it's assigned to
- You can call methods on nil receivers (careful: they can panic or they can be designed to work)
- A nil slice has length 0 and works with `append`, `len`, and `range`
- A nil map *panics* on write but returns zero values on read

```go
var s []int      // nil
len(s)           // 0 — safe
s = append(s, 1) // works — creates [1]

var m map[string]int  // nil
m["key"]              // 0 — safe (returns zero value)
m["key"] = 1          // PANIC — can't write to nil map
```

---

## 8. Error Handling: The Philosophy of Explicit Failure

Go's error handling is the thing Python developers find most alien. There are no exceptions. There is no `try/except`. There is only:

```go
result, err := doSomething()
if err != nil {
    return ..., err  // or handle it
}
// use result
```

### The philosophy

In Python, exceptions are "ask forgiveness, not permission" — you attempt something and handle failure in a separate block. In Go, errors are **values** that flow through the same channels as results. The philosophy is: failure is a normal part of program execution, not an exceptional event that needs special syntax.

This means every error is handled at the call site. You can't accidentally swallow an error (the compiler warns about unused variables). You can't have an exception silently propagate up five stack frames to a generic `except Exception:` handler.

The tradeoff: Go code has more `if err != nil` checks. Proponents say this forces you to think about every failure mode. Critics call it boilerplate.

### Error wrapping with `fmt.Errorf` and `%w`

```go
// auth/oauth.go:28-30
code, err := getOOBCode()
if err != nil {
    return "", fmt.Errorf("getting OOB code: %w", err)
}
```

The `%w` verb **wraps** the original error, creating a chain. This is analogous to Python's `raise NewError("context") from original_error`. The caller can later unwrap it with `errors.Is()` or `errors.As()` to check for specific error types deep in the chain.

A wrapped error produces messages like:

```
getting OOB code: unexpected status 401: Unauthorized
```

The context builds up as the error propagates through call layers — each layer adds its own context.

### Inspecting error types

```go
// config/config.go:31-34
if os.IsNotExist(err) {
    return &Config{}, nil
}
```

This inspects the error: is it a "file not found"? It makes a policy decision: a missing config file isn't an error, it means "use defaults." This is Go's version of:

```python
try:
    data = open("config.json").read()
except FileNotFoundError:
    return Config()
```

### The `_` discard for intentionally ignored errors

```go
// auth/oauth.go:36
_ = browser.OpenURL(approveURL)
```

The blank identifier `_` explicitly discards the return value. Opening the browser might fail (headless server, no display), but the code doesn't care — it already printed the URL as a fallback. In Python, you'd just call the function without capturing the return. In Go, some linters warn about unchecked errors, so `_ =` is an explicit "I know this can fail, and I'm OK with it."

> **Reference:** `config/config.go:24-41` (error handling chain), `auth/oauth.go:28-30` (error wrapping), `auth/oauth.go:36` (intentional discard)

---

## 9. Value vs Pointer Semantics: Who Owns the Data?

This is subtle, important, and has no direct Python equivalent. In Python, everything is a reference — you never think about this. In Go, you choose: does a function receive a **copy** (value) or a **reference** (pointer)?

### The `&` and `*` operators

```go
cfg := Config{OAuthToken: "abc"}  // value
ptr := &cfg                        // &takes the address → *Config pointer
token := ptr.OAuthToken            // Go auto-dereferences: same as (*ptr).OAuthToken
```

Python equivalent:

```python
# Everything in Python is a reference — there's no equivalent decision
cfg = Config(oauth_token="abc")    # cfg is already a reference
ptr = cfg                          # ptr points to the same object
```

### Value receivers in teleput

```go
// ui/model.go:36
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    m.width = msg.Width   // modifying the copy
    return m, nil         // returning the modified copy
}
```

`(m Model)` is a **value receiver**. The method receives a *copy* of the Model. Modifications to `m` inside the method are modifications to the copy, not the original.

"Wait," you think, "then how does Update modify the model?" It modifies the copy and **returns it.** The Bubble Tea runtime replaces the old model with the returned one. This is the Elm Architecture's approach: never mutate state in place, return new state.

### Pointer receivers (not in teleput, but fundamental)

```go
// Hypothetical — NOT in teleput
func (m *Model) SetWidth(w int) {
    m.width = w  // modifies the original
}
```

The `*` before `Model` makes this a pointer receiver. `m` is a pointer, and modifications affect the original struct.

### The decision heuristic

| Use value receiver when... | Use pointer receiver when... |
|---|---|
| The method doesn't modify the struct | The method needs to modify state |
| The struct is small (< ~64 bytes) | The struct is large (expensive to copy) |
| You want immutability guarantees | You need shared mutable state |
| Implementing functional patterns (Elm) | Implementing stateful objects |

Teleput uses **value receivers exclusively** because the Elm Architecture demands immutability. Every `Update` returns a new model.

### Why `*putio.Client` is a pointer

```go
// ui/model.go:13
client *putio.Client
```

The client holds HTTP connection pools, internal state, etc. Copying it would be wasteful and incorrect (two copies sharing the same TCP connections could interfere). Using a pointer means "everyone shares one client instance." Both `Model` and `browserModel` hold the same pointer.

> **Reference:** `ui/model.go:36-58` (value receiver returning modified copy), `ui/model.go:13` (pointer field)

---

## 10. Slices, Maps, and Collection Semantics

### Slices: the internals

A Go slice is a **three-word header** on the stack: (pointer to backing array, length, capacity).

```
slice header: [ ptr | len=3 | cap=5 ]
                 ↓
backing array: [ a | b | c | _ | _ ]
                              ^unused capacity
```

This is like Python's `list` under the hood, except the header is a value type — copying a slice copies the header, not the data. Two slices can share the same backing array.

### `append` creates a new header

```go
// ui/browser.go:79
m.parents = append(m.parents, m.parentID)
```

`append` might return a **different slice header** (if the backing array needed to grow). This is why you must always write `s = append(s, x)`, not just `append(s, x)`.

```go
// This is a BUG:
append(m.parents, m.parentID)  // return value discarded!

// This is correct:
m.parents = append(m.parents, m.parentID)
```

In Python, `list.append()` mutates in place and returns `None`. In Go, `append` returns a new slice header and the old one might be stale.

### Slice operations: no negative indexing

```go
// ui/browser.go:85-86  (stack pop)
prev := m.parents[len(m.parents)-1]
m.parents = m.parents[:len(m.parents)-1]
```

Go doesn't have negative indexing. `parents[-1]` is a compile error. You must write `parents[len(parents)-1]`.

Slice expressions use the `[low:high]` syntax (same as Python), but:
- `s[2:5]` — elements 2, 3, 4 (same as Python)
- `s[:3]` — first 3 elements (same as Python)
- `s[2:]` — from element 2 to end (same as Python)
- There is no `s[-1]` or `s[::2]` (step) — Go keeps slicing simple

### `nil` slices are usable

```go
var parents []int64  // nil slice
len(parents)         // 0
parents = append(parents, 42)  // works! → [42]
```

A nil slice behaves like an empty slice for `len`, `cap`, `append`, and `range`. You don't need `make([]int64, 0)` or `[]int64{}`. The zero value is useful.

### Maps (not in teleput, but foundational)

Go maps are hash tables, like Python dicts:

```go
m := map[string]int{
    "one":   1,
    "two":   2,
    "three": 3,
}
v, ok := m["one"]  // v=1, ok=true
v, ok = m["four"]  // v=0 (zero value), ok=false
```

The two-value lookup `v, ok := m[key]` is Go's equivalent of `dict.get()` with existence checking. The `ok` idiom is pervasive in Go.

> **Reference:** `ui/browser.go:79` (append), `ui/browser.go:85-86` (slice truncation), `ui/browser.go:20` (nil slice field)

---

## 11. Closures and First-Class Functions

Go functions are first-class values, just like in Python. You can assign them to variables, pass them as arguments, and return them from functions.

### Function types

```go
// tea.Cmd is defined as:
type Cmd func() Msg
```

A `Cmd` is a function that takes nothing and returns a `Msg`. It's a **type alias for a function signature**. In Python terms:

```python
Cmd = Callable[[], Msg]
```

### Closures capture their environment

```go
// ui/browser.go:37-53
func (m browserModel) loadDir(parentID int64) tea.Cmd {
    return func() tea.Msg {
        // captures m.client and parentID from the enclosing scope
        children, _, err := m.client.Files.List(context.Background(), parentID)
        if err != nil {
            return errMsg{err}
        }
        // ...
        return filesLoadedMsg{files: children, parentID: parentID}
    }
}
```

The returned function is a **closure** — it captures `m.client` and `parentID` from the enclosing scope. The closure outlives the function that created it (Bubble Tea stores it and calls it later, in a goroutine).

This is identical to Python closures:

```python
def load_dir(self, parent_id: int) -> Callable[[], Msg]:
    client = self.client  # captured
    def cmd() -> Msg:
        children = client.files.list(parent_id)  # uses captured variable
        return FilesLoadedMsg(files=children, parent_id=parent_id)
    return cmd
```

### Variable capture: value vs reference

**Critical Go gotcha** — closures capture variables **by reference** (actually, by reference to the variable's location), not by value:

```go
// CLASSIC BUG (not in teleput):
for i := 0; i < 5; i++ {
    go func() {
        fmt.Println(i)  // i is captured by reference — always prints 5!
    }()
}

// FIX: pass as parameter
for i := 0; i < 5; i++ {
    go func(i int) {
        fmt.Println(i)  // i is a copy — prints 0, 1, 2, 3, 4
    }(i)
}
```

Python has the same gotcha with `lambda`:

```python
# Same bug in Python!
funcs = [lambda: i for i in range(5)]
[f() for f in funcs]  # [4, 4, 4, 4, 4]

# Fix: default argument capture
funcs = [lambda i=i: i for i in range(5)]
```

Teleput avoids this because it never launches goroutines in a loop with a closure over the loop variable.

### Nested closures: the sort comparator

```go
// ui/browser.go:43
sort.Slice(children, func(i, j int) bool {
    // captures `children` from enclosing closure
})
```

This is a closure *inside* a closure — the sort comparator captures `children`, which was itself obtained inside the `loadDir` closure. Python equivalent: a lambda inside a lambda, both capturing from outer scopes.

> **Reference:** `ui/browser.go:37-53` (closure as command), `ui/browser.go:43-50` (nested closure)

---

## 12. Defer, Panic, and Resource Management

### `defer`: cleanup without context managers

```go
// auth/oauth.go:47-48
resp, err := http.Get(BaseURL + "/oauth2/oob/code?app_id=" + ClientID)
if err != nil {
    return "", err
}
defer resp.Body.Close()
```

`defer` schedules a function call to execute **when the surrounding function returns** — regardless of how it returns. It's Go's `finally` clause, but scoped to a single line.

Python equivalents:

```python
# Context manager (Pythonic)
with open("file") as f:
    data = f.read()

# Or manually:
f = open("file")
try:
    data = f.read()
finally:
    f.close()
```

Go doesn't have context managers (`with` statements), so `defer` fills that role. The advantage: `defer` is placed right next to the resource acquisition, making it visually obvious that cleanup is handled. The `with` statement is arguably more structured, but `defer` is more flexible — you can defer any function call, not just objects implementing `__enter__`/`__exit__`.

### Defer ordering: LIFO

Multiple defers execute in **last-in, first-out** order:

```go
defer fmt.Println("first")
defer fmt.Println("second")
defer fmt.Println("third")
// Output when function returns: "third", "second", "first"
```

This LIFO ordering is intentional: resources acquired later depend on resources acquired earlier, so they should be released first.

### The ticker cleanup pattern

```go
// auth/oauth.go:66-67
ticker := time.NewTicker(2 * time.Second)
defer ticker.Stop()
```

`time.NewTicker` creates an internal goroutine that sends on a channel every 2 seconds. If you don't call `Stop()`, that goroutine leaks forever. `defer ticker.Stop()` ensures cleanup regardless of how `pollForToken` exits — timeout, context cancellation, or success.

### Panic and recover (not in teleput)

Go has `panic()` (like Python's `raise` for unrecoverable errors) and `recover()` (like a last-resort `except` in a `defer`). Panics are for programming errors (nil pointer dereference, index out of bounds), not for expected failures. Teleput never panics explicitly because all failures are handled as error values.

```go
// Illustrative — NOT in teleput
func safeDivide(a, b int) (result int, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("panic: %v", r)
        }
    }()
    return a / b, nil  // if b == 0, this panics, caught by recover()
}
```

> **Reference:** `auth/oauth.go:47-48` (defer Close), `auth/oauth.go:66-67` (defer Stop)

---

## 13. Concurrency: Goroutines, Channels, and Select

This is Go's most distinctive feature set, and `auth/oauth.go` contains teleput's most interesting concurrency code.

### Goroutines: the lightweight concurrency primitive

A goroutine is a function executing concurrently with other goroutines in the same address space. Key characteristics:

- **Start cost**: ~2-4KB of stack. A Python thread needs ~8MB. You can run millions of goroutines.
- **Scheduling**: M:N — the Go runtime multiplexes M goroutines onto N OS threads. In Python, the GIL means only one thread executes Python code at a time.
- **No `async/await`**: Regular blocking code runs on goroutines. The runtime transparently parks a goroutine when it blocks on I/O and schedules another one. This is like if Python had an event loop that worked with synchronous code.

```go
go func() {
    result := expensiveOperation()  // blocking — no await needed
    ch <- result
}()
```

### Channels: typed communication pipes

A **channel** is a typed conduit for sending values between goroutines. Think of it as `queue.Queue[T]` from Python, but built into the language:

```go
ch := make(chan int)      // unbuffered — send blocks until someone receives
ch := make(chan int, 10)  // buffered — up to 10 values can queue

ch <- 42     // send
v := <-ch    // receive
```

**Unbuffered channels** synchronise: the sender blocks until a receiver is ready, and vice versa. This is like two people handing off a baton — the handoff is the synchronisation point.

**Buffered channels** decouple: the sender only blocks when the buffer is full. Like a mailbox with limited capacity.

### The `select` statement: channel multiplexing

`select` is Go's most powerful concurrency primitive. It blocks until one of several channel operations can proceed:

```go
// auth/oauth.go:71-87
for {
    select {
    case <-ctx.Done():
        return "", ctx.Err()
    case <-timeout:
        return "", fmt.Errorf("authentication timed out after 5 minutes")
    case <-ticker.C:
        token, err := checkCode(code)
        if err != nil {
            continue
        }
        if token != "" {
            return token, nil
        }
    }
}
```

Read this as: "Loop forever. On each iteration, block until one of three channels has a value:

1. **`ctx.Done()`** — The context was cancelled (e.g., Ctrl+C). Return the cancellation error.
2. **`timeout`** — 5 minutes elapsed. Return a timeout error.
3. **`ticker.C`** — 2 seconds elapsed. Check if the user approved the OAuth flow.

The `select` races the three channels against each other. Whichever fires first wins."

This is fundamentally different from anything in Python. The closest equivalent:

```python
async def poll_for_token(code: str) -> str:
    async with asyncio.timeout(300):
        while True:
            await asyncio.sleep(2)
            token = check_code(code)
            if token:
                return token
```

But the Go version handles all three "exit conditions" (cancel, timeout, tick) symmetrically in one `select` block. In Python's asyncio, cancellation comes from `Task.cancel()`, timeout from `asyncio.timeout`, and the polling interval from `asyncio.sleep` — three different mechanisms. Go's `select` unifies them.

### Key `select` semantics

- If multiple cases are ready simultaneously, one is chosen **at random** (non-deterministic).
- A `default` case makes `select` non-blocking:
  ```go
  select {
  case msg := <-ch:
      // got a message
  default:
      // ch had nothing — don't block
  }
  ```
- `select {}` (no cases) blocks forever. This is sometimes used in `main()` of server programs to keep the process alive.

### Where concurrency hides in teleput

You don't see `go` keyword or `make(chan ...)` in teleput's own code (other than the auth polling). That's because **Bubble Tea manages concurrency for you**. When `Update` returns a `tea.Cmd`, the runtime does:

```go
go func() {
    msg := cmd()       // execute the command in a goroutine
    p.msgs <- msg      // send the result on an internal channel
}()
```

The runtime then receives messages from this channel and calls `Update`. This is the Elm Architecture's command system — side effects happen in the runtime's goroutines, and results come back as messages.

> **Reference:** `auth/oauth.go:65-87` (select, channels, ticker)

---

## 14. Context: Cancellation, Deadlines, and Request Scope

### What `context.Context` is

`context.Context` is an interface that carries:
- **Cancellation signals** — "stop what you're doing"
- **Deadlines** — "stop after this time"
- **Values** — request-scoped key-value pairs

It's designed to propagate through function call chains, so that cancelling a top-level operation automatically cancels all sub-operations.

### The context hierarchy

```go
ctx := context.Background()                    // root — never cancels
ctx, cancel := context.WithCancel(parent)      // cancellable child
ctx, cancel := context.WithTimeout(parent, 5*time.Second)  // auto-cancel after 5s
ctx, cancel := context.WithDeadline(parent, time.Now().Add(5*time.Second))  // same
```

Each child context cancels when its parent does. This creates a **tree of cancellation**: cancelling the root cancels everything.

### Context in teleput

```go
// main.go:40
token, err = auth.Authenticate(context.Background())
```

`context.Background()` is the root context — no cancellation. In production, you'd derive a cancellable context so Ctrl+C during OAuth is clean.

```go
// auth/oauth.go:27
func Authenticate(ctx context.Context) (string, error) {
```

Convention: **context is always the first parameter.** Always. This is enforced by linters and code review. The function doesn't create a context — it *receives* one from the caller. This is dependency injection for cancellation.

```go
// ui/browser.go:39
children, _, err := m.client.Files.List(context.Background(), parentID)
```

Using `context.Background()` here means API calls can't be cancelled. A more sophisticated app would store a cancellable context and pass it to API calls, so navigating away cancels in-flight requests.

### Python comparison

Python's `contextvars` (3.7+) solves a different problem: task-local storage (like thread-local variables). Go's `context` is closer to a combination of:

- `asyncio.Task.cancel()` — for cancellation propagation
- Explicit timeout parameters — for deadlines
- A key-value store — for request-scoped data (like Flask's `g`)

> **Reference:** `main.go:40`, `auth/oauth.go:27`, `auth/oauth.go:71-73`, `ui/browser.go:39`

---

## 15. Visibility, Constants, and the Package-Level Namespace

### Capitalisation as access control

Go has exactly two visibility levels:

| First letter | Visibility | Python equivalent |
|---|---|---|
| Uppercase (`NewModel`, `Config`) | Exported (public) | No underscore |
| Lowercase (`newBrowserModel`, `errMsg`) | Unexported (package-private) | `_single_underscore` |

There is no "protected" or "friend" visibility. Everything is either public to the world or private to the package.

### Teleput's visibility decisions

**Exported** (used from outside `package ui` or `package config`):

```go
func NewModel(token string) Model       // ui/model.go:21 — called from main.go
type Model struct { ... }                // ui/model.go:12 — used by tea.NewProgram
func Load() (*Config, error)            // config/config.go:24 — called from main.go
func Authenticate(ctx) (string, error)  // auth/oauth.go:27 — called from main.go
```

**Unexported** (internal to the package):

```go
type browserModel struct { ... }        // ui/browser.go:15 — only used within ui
func newBrowserModel(client) browserModel  // ui/browser.go:31
type errMsg struct{ err error }          // ui/model.go:71 — only used within ui
type keyMap struct { ... }               // ui/model.go:73
var keys = keyMap{ ... }                 // ui/model.go:78
```

### Struct field visibility and JSON

```go
type Config struct {
    OAuthToken string `json:"oauth_token"`  // Exported — capital O
}
```

`OAuthToken` must be exported for `json.Unmarshal` to work — the JSON package uses reflection, and reflection can only access exported fields. If this were `oAuthToken`, JSON serialisation would silently ignore it.

### Constants

```go
// auth/oauth.go:14-17
const (
    ClientID = "8918"
    BaseURL  = "https://api.put.io/v2"
)
```

Go constants are **compile-time** values — they don't have an address in memory. You cannot take the address of a constant, pass it as a pointer, or modify it. This is like Python's `UPPER_CASE` convention, but enforced.

### Package-level `var` blocks

```go
// ui/theme.go:6-16
var (
    catMauve    = lipgloss.Color("#cba6f7")
    catPeach    = lipgloss.Color("#fab387")
    // ...
)

// ui/model.go:78-84
var keys = keyMap{
    Up:    key.NewBinding(key.WithKeys("up", "k")),
    // ...
}
```

Package-level `var` declarations are initialised once when the package loads. They're like Python's module-level variables. The `var (...)` block groups related declarations.

> **Reference:** `ui/model.go:12-19` (unexported fields), `config/config.go:11-13` (exported field), `auth/oauth.go:14-17` (constants)

---

## 16. Dependency Management: go.mod and the Module System

### go.mod structure

```go
// go.mod:1-11
module github.com/jack/teleput

go 1.25.5

require (
    github.com/charmbracelet/bubbletea v1.3.10
    github.com/charmbracelet/lipgloss v1.1.0
    github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
    github.com/putdotio/go-putio v1.7.2
    golang.org/x/oauth2 v0.35.0
)
```

| go.mod concept | Python equivalent |
|---|---|
| `module github.com/jack/teleput` | `[project] name = "teleput"` |
| `go 1.25.5` | `requires-python = ">=3.12"` |
| `require (...)` | `[project] dependencies` |
| `// indirect` | Transitive deps (in uv.lock) |
| `go.sum` | `uv.lock` — content-addressed hashes |

### Pseudo-versions

```go
github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
```

This is a **pseudo-version**: the package has no semver tags, so Go generates a version from the commit date and hash. It's like pinning `browser @ git+https://...@5ac0b6a` in pip.

### The module cache

Go modules are cached globally in `$GOPATH/pkg/mod` (default `~/go/pkg/mod`). There's no per-project virtual environment. Each module version is stored once and shared across all projects. This is more like `uv`'s global cache.

### Major version paths

```go
import "github.com/aymanbagabas/go-osc52/v2"  // v2
```

Go enforces semver for module paths. Major version `v2+` must appear in the import path. Different major versions can coexist in the same binary. Python has no equivalent.

> **Reference:** `go.mod` (complete dependency manifest)

---

## 17. Memory Model: Stack, Heap, and Escape Analysis

### Stack vs heap

In Python, almost everything lives on the heap, managed by reference counting + GC. In Go, the compiler decides:

- **Stack**: Fast, automatic. Local variables that don't outlive their function.
- **Heap**: Slower, GC-managed. Values that escape the function (returned pointers, closed-over variables, interface values).

### Escape analysis

The Go compiler performs **escape analysis** to determine whether a variable can live on the stack or must be moved to the heap:

```go
func newConfig() *Config {
    cfg := Config{OAuthToken: "abc"}
    return &cfg  // cfg ESCAPES — must be heap-allocated
}

func processConfig() {
    cfg := Config{OAuthToken: "abc"}
    fmt.Println(cfg.OAuthToken)  // cfg does NOT escape — stack-allocated
}
```

You can see escape analysis decisions with `go build -gcflags="-m"`:

```
./config.go:48: &Config{...} escapes to heap
```

### Why this matters for teleput

In `ui/browser.go:125-127`:

```go
dirSt := lipgloss.NewStyle().Bold(true).Foreground(catSapphire)
normalSt := lipgloss.NewStyle().Foreground(catText)
sizeSt := lipgloss.NewStyle().Foreground(catOverlay1)
```

These styles are created on every `view()` call. In Python, you'd worry about allocation cost. In Go, `lipgloss.Style` is a small struct — escape analysis determines it stays on the stack, so creation is essentially free (just setting some bytes in the stack frame).

### The garbage collector

Go's GC is concurrent tri-color mark-and-sweep. Key properties:

- **Concurrent**: GC runs alongside your goroutines, not stop-the-world.
- **Low latency**: Targets sub-millisecond pauses. For a TUI that redraws at 60fps (~16ms frames), GC pauses are invisible.
- **Not reference-counted**: No reference cycles problem. No `gc.collect()` needed.

This is one of Go's biggest wins for applications like TUIs where consistent frame timing matters.

---

# Part II: Terminal UI Foundations

## 18. How Terminals Actually Work

To understand Bubble Tea and Lip Gloss, you need to understand what a terminal actually is. This is knowledge most developers never acquire because frameworks abstract it away.

### The terminal is a character grid

A terminal is fundamentally a **grid of cells**, each containing a character and attributes (colour, bold, underline). When teleput runs, the entire UI is just characters written to this grid. There are no pixels, no GPU rendering, no layout engine. Just characters.

```
┌──────────────────────────────┐
│ teleput                      │  ← row 1: title bar
│ > 📁 Movies                  │  ← row 2: cursor + folder
│   📁 Music                   │  ← row 3: folder
│   📄 readme.txt  1.2 KB      │  ← row 4: file + size
│                              │
│  3 items                     │  ← last row: status bar
└──────────────────────────────┘
```

### Raw mode vs cooked mode

By default, terminals operate in **cooked mode** (line mode): input is buffered until Enter, and special characters (Ctrl+C, backspace) are handled by the terminal driver. This is what your shell uses.

Bubble Tea switches to **raw mode**: every keypress is delivered immediately to the program, and no special handling occurs. This is how teleput can respond to individual `j`/`k` keypresses without waiting for Enter.

Python equivalents:
- `curses.initscr()` does the same mode switch
- `tty.setraw(sys.stdin)` for manual raw mode
- `blessed` and `textual` handle this internally

### The alternate screen buffer

```go
// main.go:54
p := tea.NewProgram(m, tea.WithAltScreen())
```

Terminals have two screen buffers:

1. **Normal buffer**: Where your shell prompt, command output, and scrollback live.
2. **Alternate buffer**: A separate canvas that programs can switch to.

`vim`, `less`, `top`, and teleput all use the alternate screen. When they exit, the terminal switches back to the normal buffer and your shell output is untouched. Without the alternate screen, teleput would render inline and leave its output in your scrollback.

The switch is done via ANSI escape sequences:
- `\033[?1049h` — switch to alternate screen
- `\033[?1049l` — switch back to normal screen

Bubble Tea's `WithAltScreen()` handles this automatically.

### Terminal size

When a terminal is resized, the running program receives a `SIGWINCH` signal (Unix) or equivalent event. Bubble Tea catches this and sends a `tea.WindowSizeMsg`:

```go
// ui/model.go:38-43
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    m.browser.width = msg.Width
    m.browser.height = msg.Height
```

This is why teleput needs to track width and height — the terminal can be resized at any time, and the UI must adapt.

---

## 19. ANSI Escape Sequences and Styled Output

### What Lip Gloss actually does

When you write:

```go
lipgloss.NewStyle().Bold(true).Foreground(catSapphire).Render("Movies")
```

Lip Gloss produces a string like:

```
\033[1m\033[38;2;116;199;236mMovies\033[0m
```

Broken down:
- `\033[1m` — enable bold
- `\033[38;2;116;199;236m` — set foreground to RGB(116, 199, 236) (catSapphire)
- `Movies` — the actual text
- `\033[0m` — reset all attributes

The terminal interprets these escape sequences and renders the text accordingly. Lip Gloss is essentially a string formatter that wraps text in the right escape codes.

### Colour depth

Terminals support different colour depths:
- **16 colours**: The classic ANSI palette
- **256 colours**: Extended palette
- **True colour (24-bit)**: 16 million colours via RGB values

Teleput uses hex colours (`"#cba6f7"`), which Lip Gloss renders as 24-bit RGB when the terminal supports it, and downgrades to 256 or 16 colours otherwise. The `charmbracelet/colorprofile` dependency handles this detection.

### String width vs byte length

A tricky problem in terminal UIs: how wide is a string? `"📁"` is 4 bytes in UTF-8 but occupies **2 cells** in most terminals (it's a wide character). CJK characters also occupy 2 cells. ASCII characters occupy 1.

Lip Gloss handles this via the `go-runewidth` and `displaywidth` dependencies (visible in `go.mod`). When you call `lipgloss.Width(s)`, it returns the **display width** in terminal cells, not the byte length or rune count.

This is why `lipgloss.PlaceHorizontal(m.width, lipgloss.Center, loading)` works correctly even with emoji — it calculates padding based on display width.

---

## 20. The Event Loop: From Raw Input to Application State

### What an event loop is

Every interactive application has an event loop — a cycle of "wait for input, process it, update display." This is true for GUIs (Qt, GTK), web browsers (the browser event loop), game engines (the game loop), and TUIs.

```
         ┌──────────────────────────────┐
         │          Event Loop           │
         │                              │
    ┌────┴────┐    ┌──────────┐    ┌───┴───┐
    │  Wait   │───>│ Process  │───>│Render │───┐
    │for input│    │  event   │    │       │   │
    └─────────┘    └──────────┘    └───────┘   │
         ▲                                      │
         └──────────────────────────────────────┘
```

### How Bubble Tea's event loop works

Bubble Tea's loop is more sophisticated because it separates side effects from state updates:

```
    ┌─────────────────────────────────────────────────┐
    │               Bubble Tea Runtime                │
    │                                                 │
    │  ┌──────┐     ┌────────┐     ┌──────┐          │
    │  │ Msgs │────>│ Update │────>│ View │──> Screen │
    │  │Queue │     │(model) │     │(str) │           │
    │  └──┬───┘     └───┬────┘     └──────┘          │
    │     ▲             │ Cmd                         │
    │     │         ┌───▼────┐                        │
    │     └─────────│Goroutin│ (async side effects)   │
    │               │  Pool  │                        │
    │               └────────┘                        │
    └─────────────────────────────────────────────────┘
```

1. **Messages arrive** from various sources: key presses (terminal input), window resizes (OS signals), command results (goroutines completing).
2. **`Update`** receives one message at a time, produces a new model and optionally a command.
3. **Commands** are executed in goroutines. When they complete, their result (a message) enters the queue.
4. **`View`** is called after every `Update`, producing a string.
5. **The runtime diffs** the new string against the previous one and emits only the changed characters to the terminal. This is like React's virtual DOM diffing, but for text.

### Comparison to Python's event loops

| Concept | Python asyncio | Python curses | Bubble Tea |
|---|---|---|---|
| Event loop | `asyncio.get_event_loop()` | Manual `while` loop | Runtime-managed |
| Async ops | `async/await` | Threads or polling | Commands (closures) |
| State | Mutable objects | Mutable globals | Immutable model values |
| Rendering | N/A | Direct `stdscr.addstr()` | `View() -> string` diff |
| Input | N/A | `stdscr.getch()` | Messages in queue |

The key insight: Bubble Tea **inverts control**. In curses, you write the event loop. In Bubble Tea, the framework runs the loop and calls *your* functions. This is the Hollywood Principle: "don't call us, we'll call you."

---

## 21. The Elm Architecture: A General Theory of UI

### Origin and philosophy

The Elm Architecture (TEA) comes from the Elm programming language (a functional language for web UIs). Its core insight: **all UI state transitions can be modelled as a pure function from (state, event) to (new state, side effects).**

This is the same insight behind:
- **Redux** (JavaScript) — reducer functions: `(state, action) -> newState`
- **SwiftUI** — `@State` + view body recomputation
- **Flutter's BLoC** — events in, states out
- **Game engines** — update function called every frame

### The three pillars

```
Model:   The complete state of the application
Update:  (Model, Msg) → (Model, Cmd)  — pure state transition
View:    Model → String                — pure rendering
```

**Model** is a value — a snapshot of everything the application knows. In teleput:

```go
// ui/model.go:12-19
type Model struct {
    client   *putio.Client
    width    int
    height   int
    browser  browserModel   // contains files, cursor, parents, loading
    err      error
    quitting bool
}
```

**Update** is a pure function (in the functional programming sense): given the same model and message, it always produces the same result. No hidden state, no side effects. Side effects are expressed as commands — values that *describe* the effect without performing it.

**View** is also pure: given the same model, it always produces the same string. The runtime handles the messy business of actually writing to the terminal.

### Why immutability matters

In a traditional event-driven system:

```python
# Curses-style: direct mutation
def on_key_down(self):
    self.cursor += 1
    self.redraw()
```

Problems:
- If two events arrive simultaneously, they might mutate `self.cursor` concurrently.
- If `redraw()` fails halfway, the screen is inconsistent with the state.
- Testing requires mocking the entire environment.

In the Elm Architecture:

```go
// Return new state — don't mutate
case key.Matches(msg, keys.Down):
    if m.cursor < len(m.files)-1 {
        m.cursor++
    }
```

Wait — `m.cursor++` looks like mutation! But remember: `m` is a **value receiver** copy (Section 9). The original model is untouched. The modified copy is returned from `Update`, and the runtime atomically replaces the old model with the new one.

This means:
- No race conditions — there's only one `Update` call at a time.
- The previous model is available for diffing.
- Testing is trivial: create a model, call `Update` with a message, assert on the returned model.

### Commands: bridging pure and impure

The insight that makes TEA work in practice: **side effects are values, not actions.** A command doesn't *do* anything — it's a function that *will* do something when the runtime calls it.

```go
// This doesn't make an API call — it returns a function that will
return m.loadDir(parentID)
```

In Python terms, think of it as returning a coroutine object without `await`-ing it:

```python
return self.load_dir(parent_id)  # returns a coroutine, doesn't execute it
```

The runtime decides when and how to execute commands — it can batch them, run them concurrently, retry them, or cancel them. The `Update` function doesn't know or care.

---

## 22. Unidirectional Data Flow vs Traditional Event Handling

### Traditional event-driven (callback model)

```python
# Python Qt/Tkinter style
class FileBrowser:
    def __init__(self):
        self.listbox.bind("<<ListboxSelect>>", self.on_select)
        self.back_button.command = self.on_back

    def on_select(self, event):
        self.current_file = self.listbox.get(event.index)
        self.update_preview()

    def on_back(self):
        self.history.pop()
        self.load_dir(self.history[-1])
        self.update_listing()
```

Problems with this approach:
- **Scattered state changes**: Any callback can modify any state. Understanding what happens when the user clicks "back" requires reading `on_back`, `load_dir`, and `update_listing`.
- **Event ordering**: What happens if `load_dir` is slow and the user clicks "back" again before it finishes?
- **Testing**: You need to simulate events, mock widgets, and inspect widget state.

### Unidirectional data flow (Elm/Bubble Tea)

```go
// All state changes go through one function
func (m browserModel) update(msg tea.Msg) (browserModel, tea.Cmd) {
    switch msg := msg.(type) {
    case filesLoadedMsg:
        m.files = msg.files
        m.cursor = 0
        m.loading = false
        return m, nil
    case tea.KeyMsg:
        switch {
        case key.Matches(msg, keys.Back):
            if len(m.parents) > 0 {
                prev := m.parents[len(m.parents)-1]
                m.parents = m.parents[:len(m.parents)-1]
                m.loading = true
                return m, m.loadDir(prev)
            }
        }
    }
    return m, nil
}
```

Advantages:
- **Single source of truth**: All state changes are in `update`. You read one function to understand all state transitions.
- **Time-travel debugging**: Because each state is a value, you can record a sequence of `(model, msg) -> model` transitions and replay them.
- **Predictable rendering**: `view(model)` is deterministic — the same model always renders the same way.
- **Testable**: `update(model, msg)` is a pure function. No mocks needed.

### The tradeoff

Unidirectional data flow adds **indirection**. In callback-style code, "when user presses Back, load the parent directory" is one function. In TEA, it's:

1. User presses Back → `tea.KeyMsg` → `update` returns `loadDir` command
2. Runtime executes `loadDir` in goroutine → API call → `filesLoadedMsg`
3. `update` receives `filesLoadedMsg` → updates files, resets cursor

Three steps instead of one. The payoff comes with complexity: as the app grows, the single `update` function remains the only place where state changes happen, and the message queue naturally serialises concurrent events.

---

# Part III: Teleput Deep Dive

## 23. Application Lifecycle: From `main()` to Pixel

Let's trace what happens when you type `teleput` and press Enter.

### 1. Token resolution (`main.go:17-51`)

```go
tokenFlag := flag.String("token", "", "...")
flag.Parse()
token := *tokenFlag
```

The `flag` package parses CLI arguments. `flag.String` returns a `*string` (pointer) — the value is filled in by `flag.Parse()`. You dereference with `*tokenFlag`. This is different from Python's `argparse` which returns an `argparse.Namespace` object.

### 2. Client construction (`ui/model.go:21-29`)

```go
ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
httpClient := oauth2.NewClient(context.Background(), ts)
client := putio.NewClient(httpClient)
```

Three layers of abstraction, each depending only on an interface. Covered in depth in Section 31.

### 3. Program creation and run (`main.go:54-58`)

```go
p := tea.NewProgram(m, tea.WithAltScreen())
if _, err := p.Run(); err != nil {
    fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    os.Exit(1)
}
```

`tea.NewProgram` creates the event loop but doesn't start it. `p.Run()` blocks until the program exits (when `Update` returns `tea.Quit`). This is analogous to `app.exec_()` in Qt or `root.mainloop()` in Tkinter.

### 4. The event loop

Once `Run()` is called:
1. Terminal enters raw mode + alternate screen
2. `Init()` is called → returns `loadDir(0)` → runtime launches goroutine
3. Runtime sends `tea.WindowSizeMsg` → `Update` stores dimensions
4. Goroutine completes → `filesLoadedMsg` → `Update` populates file list
5. `View()` renders the file list to a string → runtime writes to terminal
6. User presses keys → `tea.KeyMsg` → `Update` moves cursor / navigates
7. User presses `q` → `Update` returns `tea.Quit` → loop exits
8. Terminal restores normal mode + normal screen buffer

---

## 24. The Configuration Cascade

Teleput resolves the OAuth token through a **four-level cascade**:

```go
// main.go:17-51
// Level 1: CLI flag
token := *tokenFlag

// Level 2: Environment variable
if token == "" {
    token = os.Getenv("PUTIO_TOKEN")
}

// Level 3: Saved config file
if token == "" {
    cfg, err := config.Load()
    // ...
    token = cfg.OAuthToken
}

// Level 4: Interactive OAuth flow
if token == "" {
    token, err = auth.Authenticate(context.Background())
    // ... save token for next time
}
```

Each level has a clear fallback. The pattern is common in CLI tools (think `git config` with `--local`, `--global`, `--system` levels).

### The config package: file I/O patterns

```go
// config/config.go:15-22
func Dir() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    dir := filepath.Join(home, ".config", appName)
    return dir, os.MkdirAll(dir, 0700)
}
```

`os.MkdirAll` is like Python's `os.makedirs(dir, exist_ok=True)`. The `0700` permission (octal — the leading `0` means octal in Go) means only the owner can access the directory. Config files use `0600` — owner read/write only.

### Struct tags for serialisation

```go
type Config struct {
    OAuthToken string `json:"oauth_token"`
}
```

The backtick string is a **struct tag** — metadata read by the `encoding/json` package via reflection. Python equivalent: Pydantic's `Field(alias="...")` or dataclass metadata.

```go
data, err := json.MarshalIndent(cfg, "", "  ")  // like json.dumps(cfg, indent=2)
json.Unmarshal(data, &cfg)                        // like json.loads(data)
```

The `&` in `json.Unmarshal(data, &cfg)` is essential — Unmarshal needs a pointer to write into the struct. Without `&`, it would receive a copy and modifications would be lost.

> **Reference:** `main.go:17-51` (cascade), `config/config.go:1-53` (complete config package)

---

## 25. OAuth Out-of-Band: Polling as Concurrency

The auth package implements **OOB (Out-of-Band) OAuth** — a flow designed for CLI apps that can't receive HTTP callbacks.

### The flow

```
1. App requests OOB code from put.io         (getOOBCode)
2. App opens browser with approval URL        (Authenticate)
3. User approves in browser                   (human action)
4. App polls put.io for token every 2 seconds (pollForToken)
5. When token arrives, OAuth is complete       (checkCode returns token)
```

### The polling loop is the concurrency showcase

```go
// auth/oauth.go:65-87
func pollForToken(ctx context.Context, code string) (string, error) {
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()

    timeout := time.After(5 * time.Minute)

    for {
        select {
        case <-ctx.Done():
            return "", ctx.Err()
        case <-timeout:
            return "", fmt.Errorf("authentication timed out after 5 minutes")
        case <-ticker.C:
            token, err := checkCode(code)
            if err != nil {
                continue
            }
            if token != "" {
                return token, nil
            }
        }
    }
}
```

Key details:

- `time.NewTicker` returns a struct with a `C` field — a channel that receives the current time every 2 seconds.
- `time.After` returns a channel that fires once after 5 minutes.
- `select` blocks until one of the three channels has a value.
- `continue` inside a `select` case goes back to the `for` loop, which re-enters `select`.
- `defer ticker.Stop()` ensures the ticker's internal goroutine is cleaned up.

### HTTP requests in Go

```go
// auth/oauth.go:43-62
resp, err := http.Get(BaseURL + "/oauth2/oob/code?app_id=" + ClientID)
if err != nil {
    return "", err
}
defer resp.Body.Close()

body, err := io.ReadAll(resp.Body)
```

In Python, `requests.get(url).json()` does everything in one line. In Go, you:
1. Make the request, check for network errors
2. Defer closing the response body (it's an `io.ReadCloser` — must be closed to free the connection)
3. Read the body bytes
4. Unmarshal JSON manually

More verbose, but each step is explicit and each error is handled.

> **Reference:** `auth/oauth.go:1-113` (complete auth package)

---

## 26. The Model Layer: State as a Value

### The top-level Model

```go
// ui/model.go:12-19
type Model struct {
    client   *putio.Client
    width    int
    height   int
    browser  browserModel
    err      error
    quitting bool
}
```

This struct is the **entire application state.** Everything the UI needs to render is here or reachable from here. There are no global variables, no singletons, no side-channel state.

### Init: the first command

```go
func (m Model) Init() tea.Cmd {
    return m.browser.loadDir(0)
}
```

`Init` returns a command, not a message. The command (a closure) will be executed by the runtime in a goroutine. The result (`filesLoadedMsg` or `errMsg`) will arrive in `Update` later.

### Update: the state machine

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        // store dimensions, return nil (no side effect)
    case tea.KeyMsg:
        // check for quit, delegate to browser
    case errMsg:
        // store error, return nil
    }
    // delegate unhandled messages to browser
    m.browser, cmd = m.browser.update(msg)
    return m, cmd
}
```

The parent model handles three concerns:
1. **Window sizing** — global concern, all components need to know
2. **Quit** — global concern, exits the application
3. **Errors** — global concern, could show an error overlay

Everything else is delegated to the browser sub-model. This is **component composition** — the same pattern used in React (parent passes props, child handles local state).

### View: pure rendering

```go
func (m Model) View() string {
    if m.quitting {
        return ""
    }
    if m.width == 0 || m.height == 0 {
        return ""
    }
    return m.browser.view()
}
```

Two guards:
- `quitting` → empty string (clean exit, no final render)
- Zero dimensions → empty string (wait for `WindowSizeMsg` before rendering)

Then it delegates entirely to the browser's view. If teleput had a header/footer/sidebar, this is where they'd be composed.

### The key bindings

```go
// ui/model.go:73-84
type keyMap struct {
    Up, Down, Enter, Back key.Binding
    Quit                  key.Binding
}

var keys = keyMap{
    Up:    key.NewBinding(key.WithKeys("up", "k")),
    Down:  key.NewBinding(key.WithKeys("down", "j")),
    Enter: key.NewBinding(key.WithKeys("enter", "l", "right")),
    Back:  key.NewBinding(key.WithKeys("backspace", "h", "left")),
    Quit:  key.NewBinding(key.WithKeys("q", "ctrl+c")),
}
```

`key.NewBinding` creates a binding that matches multiple physical keys. `key.WithKeys` is a **functional option** — a function passed to a constructor to configure it. This pattern is like Python's keyword arguments but more type-safe:

```python
# Python: keyword arguments
binding = KeyBinding(keys=["up", "k"])

# Go: functional options
binding := key.NewBinding(key.WithKeys("up", "k"))
```

Functional options are Go's idiomatic way to handle optional configuration without overloaded constructors or builder objects.

> **Reference:** `ui/model.go:12-84` (complete model + keybindings)

---

## 27. The Browser Component: Sub-Models and Delegation

### State

```go
// ui/browser.go:15-24
type browserModel struct {
    client   *putio.Client
    files    []putio.File
    cursor   int
    parentID int64
    parents  []int64    // navigation stack
    loading  bool
    width    int
    height   int
}
```

The `parents` slice is a **stack** — last-in, first-out. It tracks the directory history:

```
Navigate to Movies:  parents = [0]        parentID = 123
Navigate to 2024:    parents = [0, 123]   parentID = 456
Go back:             parents = [0]        parentID = 123
Go back:             parents = []         parentID = 0
```

### Update logic

```go
// ui/browser.go:55-93
func (m browserModel) update(msg tea.Msg) (browserModel, tea.Cmd) {
    switch msg := msg.(type) {
    case filesLoadedMsg:
        m.files = msg.files
        m.parentID = msg.parentID
        m.cursor = 0
        m.loading = false
        return m, nil

    case tea.KeyMsg:
        if m.loading {
            return m, nil  // ignore input while loading
        }
        switch {
        case key.Matches(msg, keys.Up):
            if m.cursor > 0 { m.cursor-- }
        case key.Matches(msg, keys.Down):
            if m.cursor < len(m.files)-1 { m.cursor++ }
        case key.Matches(msg, keys.Enter):
            if len(m.files) > 0 && m.files[m.cursor].IsDir() {
                m.parents = append(m.parents, m.parentID)
                m.loading = true
                return m, m.loadDir(m.files[m.cursor].ID)
            }
        case key.Matches(msg, keys.Back):
            if len(m.parents) > 0 {
                prev := m.parents[len(m.parents)-1]
                m.parents = m.parents[:len(m.parents)-1]
                m.loading = true
                return m, m.loadDir(prev)
            }
        }
    }
    return m, nil
}
```

Notable patterns:

- **Loading gate**: `if m.loading { return m, nil }` — ignores all key input while a directory is loading. This prevents double-navigation if the user hammers Enter.
- **Guard clauses**: `if len(m.files) > 0 && m.files[m.cursor].IsDir()` — only enter directories (not files).
- **Stack operations**: push with `append`, pop with slice truncation.
- **Command return**: navigating to a new directory returns a `loadDir` command. The function itself returns immediately; the API call happens later in a goroutine.

### The view

```go
// ui/browser.go:95-153 (simplified)
func (m browserModel) view() string {
    var b strings.Builder

    // Title bar
    b.WriteString(titleBarStyle.Width(m.width).Render("teleput"))
    b.WriteString("\n")

    // Loading state
    if m.loading { /* centered "Loading..." */ }

    // Empty state
    if len(m.files) == 0 { /* centered "Empty folder" */ }

    // File list
    for i, f := range m.files {
        cursor := "  "
        if i == m.cursor {
            cursor = "> "  // highlighted
        }
        if f.IsDir() {
            line = fmt.Sprintf("%s📁 %s", cursor, f.Name)
        } else {
            line = fmt.Sprintf("%s📄 %s  %s", cursor, f.Name, humanSize(f.Size))
        }
        b.WriteString(line + "\n")
    }

    // Status bar
    b.WriteString(statusBarStyle.Render(fmt.Sprintf(" %d items", len(m.files))))

    return b.String()
}
```

The `range` keyword iterates over a slice, returning `(index, value)` pairs — like Python's `enumerate()`:

```go
for i, f := range m.files { ... }  // Go
for i, f in enumerate(files): ...  // Python
```

> **Reference:** `ui/browser.go:15-172` (complete browser component)

---

## 28. Commands as Deferred Side Effects

### The command pattern in depth

```go
// ui/browser.go:37-53
func (m browserModel) loadDir(parentID int64) tea.Cmd {
    return func() tea.Msg {
        children, _, err := m.client.Files.List(context.Background(), parentID)
        if err != nil {
            return errMsg{err}
        }
        sort.Slice(children, func(i, j int) bool {
            iDir := children[i].IsDir()
            jDir := children[j].IsDir()
            if iDir != jDir {
                return iDir
            }
            return strings.ToLower(children[i].Name) < strings.ToLower(children[j].Name)
        })
        return filesLoadedMsg{files: children, parentID: parentID}
    }
}
```

This is a **factory function that returns a closure**. The closure captures:
- `m.client` — the API client (via the value receiver copy)
- `parentID` — the directory to load

The closure performs three operations:
1. API call (`m.client.Files.List`)
2. Sorting (directories first, then alphabetical)
3. Message construction

All three happen in a goroutine managed by Bubble Tea. The `Update` function sees none of this — it just receives the final `filesLoadedMsg`.

### `nil` means "no command"

```go
return m, nil  // update the model, don't run any side effect
```

A nil `Cmd` is simply skipped by the runtime. This is the "no side effect" signal.

### `tea.Quit` is a special command

```go
return m, tea.Quit  // signal the runtime to exit
```

`tea.Quit` is a pre-defined command that tells the runtime to stop the event loop. After `tea.Quit`, `p.Run()` returns and `main()` exits.

---

## 29. Type Switches: Pattern Matching for Messages

### The mechanism

```go
// ui/model.go:37
switch msg := msg.(type) {
```

This is a **type switch** — Go's pattern matching for interface types. `msg` arrives as `tea.Msg` (which is `interface{}` — any type). `msg.(type)` branches based on the concrete type, and inside each case, `msg` is **narrowed** to that type.

```go
case tea.WindowSizeMsg:
    m.width = msg.Width  // msg is WindowSizeMsg here, .Width is valid
case tea.KeyMsg:
    key.Matches(msg, keys.Quit)  // msg is KeyMsg here
case errMsg:
    m.err = msg.err  // msg is errMsg here, .err is valid
```

Python comparisons:

```python
# Python 3.10+ structural pattern matching
match msg:
    case WindowSizeMsg(width=w, height=h):
        self.width = w
    case KeyMsg() as key_msg:
        if key_matches(key_msg, QUIT):
            ...

# Traditional Python
if isinstance(msg, WindowSizeMsg):
    self.width = msg.width
elif isinstance(msg, KeyMsg):
    ...
```

The Go version is more ergonomic because the variable is automatically rebound to the narrower type — no explicit cast needed.

### The `switch` without braces and implicit break

Go's `switch` doesn't fall through by default (unlike C). Each case is independent:

```go
switch x {
case 1:
    fmt.Println("one")
    // NO fall-through — "two" is NOT printed
case 2:
    fmt.Println("two")
}
```

If you want fall-through (rare), use the `fallthrough` keyword explicitly.

### The inner `switch` in browser update

```go
// ui/browser.go:68-89
switch {
case key.Matches(msg, keys.Up):
    // ...
case key.Matches(msg, keys.Down):
    // ...
}
```

`switch` without a value is a **boolean switch** — each case is a boolean expression, and the first true one wins. It's like a chain of `if/elif`:

```python
if key_matches(msg, keys.up):
    ...
elif key_matches(msg, keys.down):
    ...
```

> **Reference:** `ui/model.go:37-54` (outer type switch), `ui/browser.go:68-89` (inner boolean switch)

---

## 30. Styling with Lip Gloss: Immutable Builders and Theming

### The fluent builder pattern

```go
// ui/theme.go:20-24
var titleBarStyle = lipgloss.NewStyle().
    Bold(true).
    Foreground(catCrust).
    Background(catMauve).
    Padding(0, 1)
```

Each method call (`.Bold(true)`, `.Foreground(...)`) returns a **new Style value** with that property set. The original is unchanged. This is an immutable builder — like how `str.replace()` in Python returns a new string.

Because styles are values (not pointers), the intermediate copies are stack-allocated and essentially free.

### Dynamic styles in `view()`

```go
// ui/browser.go:125-127
dirSt := lipgloss.NewStyle().Bold(true).Foreground(catSapphire)
normalSt := lipgloss.NewStyle().Foreground(catText)
sizeSt := lipgloss.NewStyle().Foreground(catOverlay1)
```

Created on every render. In Python, you'd hoist these to class attributes. In Go, small value types on the stack have near-zero allocation cost.

### The Catppuccin Mocha palette

```go
// ui/theme.go:6-16
var (
    catMauve    = lipgloss.Color("#cba6f7")
    catPeach    = lipgloss.Color("#fab387")
    catSapphire = lipgloss.Color("#74c7ec")
    // ...
)
```

`lipgloss.Color` is a **named type** wrapping `string`:

```go
type Color string
```

`lipgloss.Color("#cba6f7")` is a type conversion, not a function call — zero allocation, zero overhead. It exists so the type system can distinguish colours from regular strings.

### Rendering

```go
dirSt.Render(name)  // wraps `name` in ANSI escape codes for bold + sapphire
```

`Render` returns a string with ANSI escape sequences. The terminal interprets these to display styled text. Lip Gloss handles terminal capability detection (16/256/truecolour) automatically.

### Layout functions

```go
lipgloss.PlaceHorizontal(m.width, lipgloss.Center, loading)
```

This centres `loading` text within `m.width` columns by padding with spaces. It accounts for Unicode display width (emoji are 2 cells wide) via the `displaywidth` dependency.

> **Reference:** `ui/theme.go:1-30` (palette + styles), `ui/browser.go:95-153` (rendering)

---

## 31. The OAuth2 Token Source Pattern

```go
// ui/model.go:22-24
ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
httpClient := oauth2.NewClient(context.Background(), ts)
client := putio.NewClient(httpClient)
```

This three-line chain demonstrates **interface-based dependency injection** — one of Go's most powerful patterns.

### Layer 1: TokenSource interface

```go
type TokenSource interface {
    Token() (*Token, error)
}
```

`StaticTokenSource` always returns the same token. But you could swap in a `TokenSource` that:
- Refreshes tokens automatically
- Reads from a credential store
- Prompts the user interactively

The HTTP client doesn't know or care which implementation it gets.

### Layer 2: Transparent auth middleware

```go
httpClient := oauth2.NewClient(context.Background(), ts)
```

This wraps Go's `http.Client` with a **transport decorator** that adds `Authorization: Bearer <token>` to every request:

```
Request → OAuth2 Transport (adds header) → Default Transport (sends HTTP) → Server
```

This is the decorator pattern. In Python:

```python
session = requests.Session()
session.headers["Authorization"] = f"Bearer {token}"
```

### Layer 3: API client accepts any HTTP client

```go
client := putio.NewClient(httpClient)
```

The put.io client takes a standard `*http.Client`. It doesn't know about OAuth — it just makes HTTP calls. Each layer depends on an abstraction (interface), not a concrete implementation.

> **Reference:** `ui/model.go:21-29` (complete chain)

---

## 32. Sorting, Formatting, and Standard Library Patterns

### Custom sort comparators

```go
// ui/browser.go:43-50
sort.Slice(children, func(i, j int) bool {
    iDir := children[i].IsDir()
    jDir := children[j].IsDir()
    if iDir != jDir {
        return iDir  // directories before files
    }
    return strings.ToLower(children[i].Name) < strings.ToLower(children[j].Name)
})
```

`sort.Slice` takes a **less function**: return true if `i` should come before `j`. Python's `list.sort(key=...)` extracts sort keys; Go's comparator is more flexible but more verbose.

The `bool` trick: when `iDir` is `true` and `jDir` is `false`, `return iDir` means "yes, `i` (a directory) comes before `j` (a file)."

### Human-readable sizes

```go
// ui/browser.go:155-171
func humanSize(bytes int64) string {
    const (
        KB = 1024
        MB = KB * 1024
        GB = MB * 1024
    )
    switch {
    case bytes >= GB:
        return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
    case bytes >= MB:
        return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
    case bytes >= KB:
        return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
    default:
        return fmt.Sprintf("%d B", bytes)
    }
}
```

This is a **boolean switch** — each case is evaluated in order, and the first true one wins. The `const` block defines compile-time constants that reference each other.

Note the explicit `float64(bytes)` conversion. Go has no implicit numeric conversions — you can't divide `int64` by `int` and get a float. You must convert explicitly. Python handles this transparently (`5 / 2 == 2.5`).

### `strings.Builder` for view rendering

```go
var b strings.Builder
b.WriteString("text")
b.WriteString("\n")
return b.String()
```

Go equivalent of Python's `io.StringIO()` or `"".join(parts)`. The zero value is ready to use — no constructor needed.

### `fmt.Sprintf` formatting verbs

| Verb | Meaning | Python equivalent |
|---|---|---|
| `%s` | String | `%s` or `{:s}` |
| `%d` | Integer | `%d` or `{:d}` |
| `%f` | Float | `%f` or `{:f}` |
| `%.1f` | Float, 1 decimal | `{:.1f}` |
| `%v` | Default format | `repr()` (roughly) |
| `%+v` | With field names | No equivalent |
| `%#v` | Go syntax literal | No equivalent |
| `%w` | Wrap error (Errorf only) | `raise X from Y` |

> **Reference:** `ui/browser.go:43-50` (sort), `ui/browser.go:155-171` (humanSize), `ui/browser.go:100` (Builder)

---

## 33. Architectural Patterns and the Composition Root

### The dependency graph

```
main.go
  ├── flag + os.Getenv → token string
  ├── config.Load() → saved token
  ├── auth.Authenticate() → OAuth flow → token
  ├── config.Save() → persist token
  ├── ui.NewModel(token) → Model
  │     ├── oauth2 token source → HTTP client → put.io client
  │     └── browserModel (unexported sub-model)
  └── tea.NewProgram(model) → run event loop
```

### Separation of concerns

| Package | Responsibility | Knows about |
|---|---|---|
| `main` | Orchestration | All packages |
| `config` | File persistence | Only `os`, `json` |
| `auth` | OAuth flow | Only `net/http`, `time` |
| `ui` | TUI rendering + interaction | `bubbletea`, `lipgloss`, `putio` |

**Notice:** `auth` doesn't import `config`. The OAuth flow returns a token string; `main` decides to save it. `config` doesn't import `auth`. They're decoupled through `main`. This is the **composition root** pattern — the single place where all dependencies are wired together.

### The state machine

Teleput's states form an implicit FSM:

```
[Initialising] ──Init()──> [Loading Root]
[Loading Root] ──filesLoaded──> [Browsing]
[Browsing] ──Enter folder──> [Loading Subfolder]
[Loading Subfolder] ──filesLoaded──> [Browsing]
[Browsing] ──Back──> [Loading Parent]
[Loading Parent] ──filesLoaded──> [Browsing]
[Browsing] ──Quit──> [Done]
[Any] ──errMsg──> [Error]
```

Each transition is a `case` in a type switch. Each state is encoded implicitly in the struct's field values (`loading`, `files`, `parents`). There's no explicit state enum.

### What teleput teaches about Go architecture

1. **Flat is better than nested.** Three packages at the same level, no package hierarchy deeper than one level.
2. **Explicit wiring over magic.** No dependency injection framework, no service locator. `main()` constructs everything.
3. **Interfaces at boundaries.** The put.io client accepts `*http.Client` (implicit interface). The Bubble Tea runtime accepts `tea.Model` (explicit interface). Internal code uses concrete types.
4. **Small packages with clear responsibilities.** Each package does one thing and has a clear public API.
5. **Value semantics for state.** The Elm Architecture + value receivers means state is never shared or mutated unexpectedly.

---

# Appendices

## Appendix A: Python-Go Quick Reference

| Python | Go (in teleput) | File reference |
|---|---|---|
| `class Foo:` | `type Foo struct { ... }` | `ui/model.go:12` |
| `def __init__(self):` | `func NewFoo() Foo { ... }` | `ui/model.go:21` |
| `self.x` | `m.x` (value receiver) | `ui/model.go:39` |
| `from x import y` | `import "path/to/x"` then `x.Y` | `main.go:9-13` |
| `try/except` | `if err != nil { }` | `config/config.go:26` |
| `with open() as f:` | `defer f.Close()` | `auth/oauth.go:48` |
| `isinstance(x, T)` | `switch x.(type) { case T: }` | `ui/model.go:37` |
| `list.append(x)` | `s = append(s, x)` | `ui/browser.go:79` |
| `json.loads(s)` | `json.Unmarshal([]byte(s), &v)` | `config/config.go:37` |
| `json.dumps(v, indent=2)` | `json.MarshalIndent(v, "", "  ")` | `config/config.go:48` |
| `asyncio.sleep(2)` | `time.NewTicker(2 * time.Second)` | `auth/oauth.go:66` |
| `os.environ.get("K")` | `os.Getenv("K")` | `main.go:22` |
| `Path.home()` | `os.UserHomeDir()` | `config/config.go:18` |
| `"".join(parts)` | `strings.Builder` | `ui/browser.go:100` |
| `sorted(l, key=fn)` | `sort.Slice(l, less func)` | `ui/browser.go:43` |
| `enumerate(seq)` | `for i, v := range seq` | `ui/browser.go:129` |
| `argparse` | `flag` package | `main.go:17` |
| `@dataclass` | `type Foo struct` | `config/config.go:11` |
| `Protocol` (typing) | `type Foo interface` | (implicit in bubbletea) |
| `None` | `nil` | throughout |
| `lambda x: x.name` | `func(x T) string { return x.Name }` | `ui/browser.go:43` |

## Appendix B: Go Concepts Not in Teleput

### Goroutine spawning with `go`

```go
go func() {
    result := expensiveOperation()
    ch <- result
}()
```

The `go` keyword launches a function in a new goroutine.

### `sync.WaitGroup` — coordinating goroutine completion

```go
var wg sync.WaitGroup
for _, url := range urls {
    wg.Add(1)
    go func(u string) {
        defer wg.Done()
        fetch(u)
    }(url)
}
wg.Wait()  // blocks until all goroutines finish
```

This is `asyncio.gather()` for goroutines.

### `sync.Mutex` — protecting shared state

```go
var mu sync.Mutex
mu.Lock()
shared.value = newValue
mu.Unlock()
```

Not needed in teleput because the Elm Architecture eliminates shared mutable state.

### Struct embedding (composition over inheritance)

```go
type Admin struct {
    User           // embedded
    Permissions []string
}
// admin.Name works — promoted from User
```

### Generics (Go 1.18+)

```go
func Map[T, U any](s []T, f func(T) U) []U {
    result := make([]U, len(s))
    for i, v := range s {
        result[i] = f(v)
    }
    return result
}
```

### Goroutine leak detection

A goroutine that never exits leaks memory. Common sources: sending on a channel with no receiver, waiting on a cancelled context without checking. `defer ticker.Stop()` in `auth/oauth.go:67` prevents exactly this kind of leak.

### The `init()` function

Each package can have `init()` functions that run automatically when the package is loaded. Teleput doesn't use them (and most Go developers prefer explicit initialisation in `main()`).

---

## Appendix C: Exercises

To deepen your understanding, try these modifications to teleput:

1. **Add file size totals**: Show the total size of all files in the current directory in the status bar. *Exercises:* iterating over slices, `fmt.Sprintf`, modifying `view()`.

2. **Add a loading spinner**: Replace the static "Loading..." text with an animated spinner using Bubble Tea's `spinner` bubble. *Exercises:* component composition, sub-model updates, `tea.Cmd` chaining, tick-based animations.

3. **Add search/filtering**: Let the user type to filter the file list. *Exercises:* text input handling, `textinput` bubble, slice filtering, managing focus between components.

4. **Add a breadcrumb bar**: Show the current path (e.g., `Your Files > Movies > 2024`). *Exercises:* maintaining additional state (directory names alongside IDs), modifying the navigation stack.

5. **Add download initiation**: When the user presses `d` on a file, start a download. *Exercises:* new key bindings, new `tea.Cmd`, new message types, goroutine-based I/O, progress tracking.

6. **Add cancellable navigation**: Pass a `context.WithCancel` to API calls so navigating to a new directory cancels the in-flight request for the old one. *Exercises:* context propagation, cancellation, storing cancel functions.

7. **Add a help overlay**: Press `?` to toggle a help screen showing keybindings. *Exercises:* state management (overlay vs normal mode), conditional rendering, `lipgloss.Place` for centred overlays.

8. **Write tests**: Test the `update` function by constructing models, sending messages, and asserting on the returned model. *Exercises:* Go testing (`go test`), table-driven tests, testing pure functions.

---

*This document was written as a learning companion to the teleput source code. All file references are relative to the project root. Read the code alongside this document — the two are designed to be used together.*
