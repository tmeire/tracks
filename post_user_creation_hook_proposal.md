# Proposal: Post-User-Creation Hook for Tracks Authentication Module

## 1. Objective
Introduce a reliable mechanism for external modules and application logic to intercept the user creation process in the `tracks` authentication module. This enables automated side-effects such as tenant provisioning, sending welcome emails, and synchronizing data with third-party CRM tools.

## 2. Technical Design

### A. Hook Signature
Define a new function type in the `authentication` module to ensure type safety and consistency across the framework.

```go
// github.com/tmeire/tracks/modules/authentication/hooks.go
type PostUserCreationHook func(ctx context.Context, user *User) error
```

### B. Hook Registry
Implement a registry within the `authentication` package to manage these hooks. This follows the pattern of existing module registrations in `tracks`.

```go
package authentication

var postUserCreationHooks []PostUserCreationHook

// OnUserCreated registers a callback to be executed after a user is created.
func OnUserCreated(hook PostUserCreationHook) {
    postUserCreationHooks = append(postUserCreationHooks, hook)
}
```

### C. Trigger Points
Update the internal user creation logic (likely within the `Registration` controller or a service layer) to execute registered hooks immediately after a successful database commit.

```go
// Example integration in the authentication module's internal logic
func (m *AuthModule) FinalizeUserCreation(ctx context.Context, user *User) error {
    for _, hook := range postUserCreationHooks {
        if err := hook(ctx, user); err != nil {
            // Log the error but allow the process to continue unless the hook is critical.
            tracks.Logger(ctx).Error("post-user-creation hook failed", "user_id", user.ID, "error", err)
        }
    }
    return nil
}
```

## 3. Transactional Considerations
*   **Context Propagation:** Hooks must receive the active `context.Context` to allow them to participate in the same database transaction if required.
*   **Error Handling:** By default, hooks should be "best-effort." If a hook failure should rollback user creation, it must be explicitly wrapped in the same transaction as the `User` record insertion.

## 4. Implementation Steps
1.  **Core Update:** Add the hook type and registry to the `authentication` module in the `tracks` repository.
2.  **Instrumentation:** Audit all user creation paths (Public Registration, Admin API, CLI) and insert the hook trigger.
3.  **Refactoring:** (Optional) If `tracks_hooks.md` is already implemented, ensure `User` model hooks are called *before* these module-level hooks.
4.  **Verification:** Add unit tests to the `authentication` module ensuring hooks are called in the correct order and with the expected data.

## 5. Proposed Use Case (Floral CRM)
Once implemented, `Floral CRM` can use this to automate tenant setup:

```go
// In Floral CRM's main.go or an init function
authentication.OnUserCreated(func(ctx context.Context, user *authentication.User) error {
    return multitenancy.ProvisionDefaultTenant(ctx, user)
})
```
