## A Load runtime2.Runtime with runtime installed by old runtime package


## B Implement the model/client/Default and model/Client/Mock and maybe add some integration test

ACs
- The functions in the files `pkg/platform/runtime2/model/*` are implemented
- A test exists ensuring the functions return expected results for a project that exists on the platforms

## C Implement the alternative Setup

ACs
- We can set up an alternative runtime with the `runtime.Setup.InstallRuntime()` method
- The installed runtime can be used with the old runtime package
- A test exists that uses the `client.Mock`.
- In this iteration the message handling does not need to work yet

## D The message handling for the new runtime is implemented

The new message handler will be very similar to the old one with two distinctions:
- more results are asynchronous: 
    - needs to support parallel processing of artifacts in different stages (download, unpack, install)
- the change message handling is generalized

ACs
- We have a struct fulfilling the `build.MessageHandler` interface
- Follow-up stories are created if the runtime setup needs to be extended (eg., because data needs to be stored for offline use)

## E Add Camel implementation

## F Ensure that everything works together

## G Remove old runtime package


# Dependencies

    A <- B <--- C <--\
             \- E <--|--- F <-- G
    D <--------------/
