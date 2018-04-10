Failures are our versions of errors and are used to:

 - differentiate between errors that are user facing and errors that are app facing (ie. us).
 - be able to infer failure type without convoluted type definitions

User errors are printed, app errors are logged. To create a failure you have to invoke `(FailureType).New(message, param1, param2)`.
The message can be a localisation key, and the params are translated to localisation params as `V0`, `V1`, etc.
The FailureType can be one of the `failures.Fail*` variables or your own defined failure type.

To define a FailureType call `failures.Type(name, parent1, parent2, ..)`. A FailureType can inherit other FailureTypes, 
which can then be used from code by checking if an error matches a type, eg. `err.Type.Matches(failures.FailUser)`.

