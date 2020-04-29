# Error Handling

Safe for custom errors, all errors should be created via the errs.New,
errs.Wrap, locale.NewError and locale.WrapError methods. This is to ensure
our error handling is properly wrapped with stacks traces.
In the long term we may remove stack traces and this requirement, but
for now this is necessary due to our legacy code not using wrapped errors.

When creating your own custom error types, always wrap them in errs.Wrap  
or locale.WrapError so that the stack traces gets added, or implement your
own `Stack()` method (see the errs.Error interface).

See errs/example_test.go for basic error usage

## Rules

Please follow these rules for all new code being written, as well as any
refactorings of old code as long as it doesn't add significant scope creep.

- Failures are deprecated, avoid them
- All new errors must satisfy the errs.Error interface
- Try to use the `errs.New`, `errs.Wrap`, `locale.NewError` and `locale.WrapError`
  methods whenever possible.
- Do not wrap errors with `%w`, we combine errors with `errs.Join()`
  (wrapping with `%w` complicates localization, and very few projects seem to actually be doing this)
- Errors should always talk about what went wrong in the current context
  don't speculate about what might have went wrong down the chain, that's
  what the wrapped errors are for
- When an error is due to user input use locale.NewInputError or locale.WrapInputError
- Errors returned from runners must always be localized
  - Any other errors should only be localized if we feel the error provides
    actionable feedback to the end-user (because non-localized errors will
    not be shown to the user)
- Localized errors must be capitalized and end with a period
- When writing localized errors you do not need to add them to en-us.yaml,
  the ID is just to source other languages once we start adding them
