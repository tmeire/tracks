# Feature Request: Form Validation Framework

**Priority:** High  
**Status:** Open

## Description

The framework lacks a structured form validation system. Currently, validation is done manually in handlers with repetitive code.

## Current Implementation

From authentication module:

```go
// Manual validation scattered in handlers
if email == "" || name == "" || password == "" || passwordConfirmation == "" {
    session.Flash(r, "alert", "All fields are required")
    return &tracks.Response{
        StatusCode: http.StatusUnprocessableEntity,
        Location:   "/users/new",
    }, nil
}

if password != passwordConfirmation {
    session.Flash(r, "alert", "Passwords do not match")
    return &tracks.Response{
        StatusCode: http.StatusUnprocessableEntity,
        Location:   "/users/new",
    }, nil
}
```

## Required Functionality

1. **Declarative Validation**: Struct tags for validation rules
2. **Built-in Validators**: Common validations (required, email, min/max length, etc.)
3. **Custom Validators**: Ability to define custom validation functions
4. **Error Aggregation**: Collect all errors, not just first failure
5. **Field-level Errors**: Associate errors with specific form fields
6. **Flash Integration**: Automatic integration with flash messages
7. **Cross-field Validation**: Compare multiple fields (e.g., password confirmation)

## Proposed API

```go
// Define validation struct
type WaitlistForm struct {
    Email    string `validate:"required,email"`
    Name     string `validate:"required,min=2,max=100"`
    Domain   string `validate:"required,fqdn"`
    Metadata string `validate:"max=500"`
}

type UserRegistrationForm struct {
    Email                string `validate:"required,email,unique=users.email"`
    Name                 string `validate:"required,min=2,max=100"`
    Password             string `validate:"required,min=8,strong_password"`
    PasswordConfirmation string `validate:"required,eqfield=Password"`
}

// In handler
func (h *Handler) Create(r *http.Request) (any, error) {
    var form WaitlistForm
    
    // Parse and validate
    if err := tracks.ParseAndValidate(r, &form); err != nil {
        // Returns validation errors that can be rendered in template
        return tracks.ValidationError(err)
    }
    
    // Form is valid, use form.Email, form.Name, etc.
}

// Custom validator
tracks.RegisterValidator("strong_password", func(value string) error {
    // Check for uppercase, lowercase, number, special char
})
```

## Use Cases

- User registration forms
- Contact/support forms
- Admin data entry
- API input validation
- Multi-step wizards

## Acceptance Criteria

- [ ] Struct tag-based validation
- [ ] Built-in validators: required, email, min, max, len, eqfield, nefield
- [ ] Custom validator registration
- [ ] Validation error aggregation
- [ ] Integration with templates (error display)
- [ ] Integration with flash messages
- [ ] Cross-field validation support
- [ ] Database uniqueness validation
- [ ] Documentation and examples

## Built-in Validators

| Validator | Description | Example |
|-----------|-------------|---------|
| required | Field must be present | `validate:"required"` |
| email | Valid email format | `validate:"email"` |
| min | Minimum length/value | `validate:"min=5"` |
| max | Maximum length/value | `validate:"max=100"` |
| len | Exact length | `validate:"len=10"` |
| eqfield | Equal to another field | `validate:"eqfield=Password"` |
| nefield | Not equal to another field | `validate:"nefield=OldPassword"` |
| fqdn | Valid domain name | `validate:"fqdn"` |
| url | Valid URL | `validate:"url"` |
| unique | Unique in database | `validate:"unique=users.email"` |
| enum | One of allowed values | `validate:"enum=admin,user,guest"` |
| regex | Matches pattern | `validate:"regex=^[a-z]+$"` |
