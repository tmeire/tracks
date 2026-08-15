
## Design guidelines

The design guidelines are listed in order of significance. This means we don't expose the raw Go http handlers, but wrap them in an interface that enable the MVC architecture. It means we take the penalty hit of the slightly slower stdlib `encoding/json` over faster third party libraries.

### "Old-school" MVC architecture

Tracks follows the MVP pattern used in popular web frameworks like Ruby on Rails, Django and Laravel.

* **M**odel (M): The data layer, responsible for storing and retrieving data
* **V**iew (V): The visualization layer, responsible for rendering the data in an usable format like HTML, XML or JSON.
* **C**ontroller (C): The logic layer, responsible for translating client requests into Models and passing those models to the View.

### The right way by default

All systems within Tracks should be setup in a way that makes the user use the recommended approach without even thinking about it. Alternative approaches should be hard to create and use.
q
> If it's hard, you're probably doing something you shouldn't be.

### Convention over configuration

Tracks relies on smart defaults and a set of fixed conventions, facilitated by an opinionated and rigid design, to keep the application and its configuration as minimal as possible. As per the guideline above, deviating from the conventions should be hard or impossible.

Even for external services, like database and cache services, the default configuration examples should be as limited as possible, even when more advanced settings are available.

### Go stdlib over custom code and libraries

Tracks defaults to Go stdlib libraries for its internals, like `net/http.ServeMux` for routing and `template/html` for rendering. It should be as easy as possible to learn Tracks, and using the stdlib concepts allows any proficient Go developer to jump right in.

### Speed over boilerplate

Tracks tries to avoid runtime performance penalties as much as possible, even if that means a bit more boilerplate. In practice, this mostly comes down to avoiding reflection wherever possible. 

### Compile-time errors over runtime errors

When all information is known at compile time and Errors in Tracks applications

### Community standards over reinvented wheels.

Tracks relies on the wider Go and software communities whenever there is an open standard that is not covered by the stdlib. Think using OpenTelemetry for tracing. There's no point in reinventing industry-wide standards, even if their Go implementations sometimes pair badly with the previous guidelines.
