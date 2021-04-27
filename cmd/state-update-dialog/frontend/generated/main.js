/*
 * ATTENTION: The "eval" devtool has been used (maybe by default in mode: "development").
 * This devtool is neither made for production nor for readable output files.
 * It uses "eval()" calls to create a separate source file in the browser devtools.
 * If you are trying to read the output file, select a different devtool (https://webpack.js.org/configuration/devtool/)
 * or disable the default devtool with "devtool: false".
 * If you are looking for production-ready output files, see mode: "production" (https://webpack.js.org/configuration/mode/).
 */
/******/ (function() { // webpackBootstrap
/******/ 	var __webpack_modules__ = ({

/***/ "./main.js":
/*!*****************!*\
  !*** ./main.js ***!
  \*****************/
/***/ (function() {

eval("// Main entry point\nfunction start() {\n  var showContextMenu = document.oncontextmenu;\n\n  document.oncontextmenu = function () {\n    return false;\n  };\n\n  backend.Bindings.DebugMode().then(function (debugMode) {\n    console.log(debugMode);\n    if (debugMode) document.oncontextmenu = showContextMenu; // Enable context menu if we're running in debug mode\n  });\n  Promise.all([backend.Bindings.CurrentVersion(), backend.Bindings.AvailableVersion()]).then(function (result) {\n    document.getElementById(\"CurrentVersion\").innerText = result[0];\n    document.getElementById(\"AvailableVersion\").innerText = result[1];\n  });\n  backend.Bindings.Warning().then(function (result) {\n    if (result === \"\") return;\n    document.getElementById(\"warning-wrapper\").style.display = \"block\";\n    document.getElementById(\"warning\").innerHTML = result;\n  });\n  populateChangelog();\n  document.getElementById(\"close-btn\").addEventListener(\"click\", function () {\n    return backend.Bindings.Exit();\n  });\n}\n\nfunction populateChangelog(tries) {\n  tries = tries || 0;\n\n  if (tries > 10) {\n    return;\n  }\n\n  backend.Bindings.Changelog().then(function (result) {\n    if (result === \"\") {\n      tries++;\n      setTimeout(populateChangelog.bind(null, tries), tries * 100);\n    } else {\n      var changelog = document.getElementById(\"changelog\");\n      changelog.style.height = \"\";\n      changelog.innerHTML = result;\n    }\n  });\n} // We provide our entrypoint as a callback for runtime.Init\n\n\nwindow.wails._.Init(start);\n\n//# sourceURL=webpack://state-update-dialog/./main.js?");

/***/ }),

/***/ "./node_modules/babel-polyfill/lib/index.js":
/*!**************************************************!*\
  !*** ./node_modules/babel-polyfill/lib/index.js ***!
  \**************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n\r\n__webpack_require__(/*! core-js/shim */ \"./node_modules/core-js/shim.js\");\r\n\r\n__webpack_require__(/*! regenerator-runtime/runtime */ \"./node_modules/babel-polyfill/node_modules/regenerator-runtime/runtime.js\");\r\n\r\n__webpack_require__(/*! core-js/fn/regexp/escape */ \"./node_modules/core-js/fn/regexp/escape.js\");\r\n\r\nif (__webpack_require__.g._babelPolyfill) {\r\n  throw new Error(\"only one instance of babel-polyfill is allowed\");\r\n}\r\n__webpack_require__.g._babelPolyfill = true;\r\n\r\nvar DEFINE_PROPERTY = \"defineProperty\";\r\nfunction define(O, key, value) {\r\n  O[key] || Object[DEFINE_PROPERTY](O, key, {\r\n    writable: true,\r\n    configurable: true,\r\n    value: value\r\n  });\r\n}\r\n\r\ndefine(String.prototype, \"padLeft\", \"\".padStart);\r\ndefine(String.prototype, \"padRight\", \"\".padEnd);\r\n\r\n\"pop,reverse,shift,keys,values,entries,indexOf,every,some,forEach,map,filter,find,findIndex,includes,join,slice,concat,push,splice,unshift,sort,lastIndexOf,reduce,reduceRight,copyWithin,fill\".split(\",\").forEach(function (key) {\r\n  [][key] && define(Array, key, Function.call.bind([][key]));\r\n});\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/babel-polyfill/lib/index.js?");

/***/ }),

/***/ "./node_modules/babel-polyfill/node_modules/regenerator-runtime/runtime.js":
/*!*********************************************************************************!*\
  !*** ./node_modules/babel-polyfill/node_modules/regenerator-runtime/runtime.js ***!
  \*********************************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("/**\r\n * Copyright (c) 2014, Facebook, Inc.\r\n * All rights reserved.\r\n *\r\n * This source code is licensed under the BSD-style license found in the\r\n * https://raw.github.com/facebook/regenerator/master/LICENSE file. An\r\n * additional grant of patent rights can be found in the PATENTS file in\r\n * the same directory.\r\n */\r\n\r\n!(function(global) {\r\n  \"use strict\";\r\n\r\n  var Op = Object.prototype;\r\n  var hasOwn = Op.hasOwnProperty;\r\n  var undefined; // More compressible than void 0.\r\n  var $Symbol = typeof Symbol === \"function\" ? Symbol : {};\r\n  var iteratorSymbol = $Symbol.iterator || \"@@iterator\";\r\n  var asyncIteratorSymbol = $Symbol.asyncIterator || \"@@asyncIterator\";\r\n  var toStringTagSymbol = $Symbol.toStringTag || \"@@toStringTag\";\r\n\r\n  var inModule = \"object\" === \"object\";\r\n  var runtime = global.regeneratorRuntime;\r\n  if (runtime) {\r\n    if (inModule) {\r\n      // If regeneratorRuntime is defined globally and we're in a module,\r\n      // make the exports object identical to regeneratorRuntime.\r\n      module.exports = runtime;\r\n    }\r\n    // Don't bother evaluating the rest of this file if the runtime was\r\n    // already defined globally.\r\n    return;\r\n  }\r\n\r\n  // Define the runtime globally (as expected by generated code) as either\r\n  // module.exports (if we're in a module) or a new, empty object.\r\n  runtime = global.regeneratorRuntime = inModule ? module.exports : {};\r\n\r\n  function wrap(innerFn, outerFn, self, tryLocsList) {\r\n    // If outerFn provided and outerFn.prototype is a Generator, then outerFn.prototype instanceof Generator.\r\n    var protoGenerator = outerFn && outerFn.prototype instanceof Generator ? outerFn : Generator;\r\n    var generator = Object.create(protoGenerator.prototype);\r\n    var context = new Context(tryLocsList || []);\r\n\r\n    // The ._invoke method unifies the implementations of the .next,\r\n    // .throw, and .return methods.\r\n    generator._invoke = makeInvokeMethod(innerFn, self, context);\r\n\r\n    return generator;\r\n  }\r\n  runtime.wrap = wrap;\r\n\r\n  // Try/catch helper to minimize deoptimizations. Returns a completion\r\n  // record like context.tryEntries[i].completion. This interface could\r\n  // have been (and was previously) designed to take a closure to be\r\n  // invoked without arguments, but in all the cases we care about we\r\n  // already have an existing method we want to call, so there's no need\r\n  // to create a new function object. We can even get away with assuming\r\n  // the method takes exactly one argument, since that happens to be true\r\n  // in every case, so we don't have to touch the arguments object. The\r\n  // only additional allocation required is the completion record, which\r\n  // has a stable shape and so hopefully should be cheap to allocate.\r\n  function tryCatch(fn, obj, arg) {\r\n    try {\r\n      return { type: \"normal\", arg: fn.call(obj, arg) };\r\n    } catch (err) {\r\n      return { type: \"throw\", arg: err };\r\n    }\r\n  }\r\n\r\n  var GenStateSuspendedStart = \"suspendedStart\";\r\n  var GenStateSuspendedYield = \"suspendedYield\";\r\n  var GenStateExecuting = \"executing\";\r\n  var GenStateCompleted = \"completed\";\r\n\r\n  // Returning this object from the innerFn has the same effect as\r\n  // breaking out of the dispatch switch statement.\r\n  var ContinueSentinel = {};\r\n\r\n  // Dummy constructor functions that we use as the .constructor and\r\n  // .constructor.prototype properties for functions that return Generator\r\n  // objects. For full spec compliance, you may wish to configure your\r\n  // minifier not to mangle the names of these two functions.\r\n  function Generator() {}\r\n  function GeneratorFunction() {}\r\n  function GeneratorFunctionPrototype() {}\r\n\r\n  // This is a polyfill for %IteratorPrototype% for environments that\r\n  // don't natively support it.\r\n  var IteratorPrototype = {};\r\n  IteratorPrototype[iteratorSymbol] = function () {\r\n    return this;\r\n  };\r\n\r\n  var getProto = Object.getPrototypeOf;\r\n  var NativeIteratorPrototype = getProto && getProto(getProto(values([])));\r\n  if (NativeIteratorPrototype &&\r\n      NativeIteratorPrototype !== Op &&\r\n      hasOwn.call(NativeIteratorPrototype, iteratorSymbol)) {\r\n    // This environment has a native %IteratorPrototype%; use it instead\r\n    // of the polyfill.\r\n    IteratorPrototype = NativeIteratorPrototype;\r\n  }\r\n\r\n  var Gp = GeneratorFunctionPrototype.prototype =\r\n    Generator.prototype = Object.create(IteratorPrototype);\r\n  GeneratorFunction.prototype = Gp.constructor = GeneratorFunctionPrototype;\r\n  GeneratorFunctionPrototype.constructor = GeneratorFunction;\r\n  GeneratorFunctionPrototype[toStringTagSymbol] =\r\n    GeneratorFunction.displayName = \"GeneratorFunction\";\r\n\r\n  // Helper for defining the .next, .throw, and .return methods of the\r\n  // Iterator interface in terms of a single ._invoke method.\r\n  function defineIteratorMethods(prototype) {\r\n    [\"next\", \"throw\", \"return\"].forEach(function(method) {\r\n      prototype[method] = function(arg) {\r\n        return this._invoke(method, arg);\r\n      };\r\n    });\r\n  }\r\n\r\n  runtime.isGeneratorFunction = function(genFun) {\r\n    var ctor = typeof genFun === \"function\" && genFun.constructor;\r\n    return ctor\r\n      ? ctor === GeneratorFunction ||\r\n        // For the native GeneratorFunction constructor, the best we can\r\n        // do is to check its .name property.\r\n        (ctor.displayName || ctor.name) === \"GeneratorFunction\"\r\n      : false;\r\n  };\r\n\r\n  runtime.mark = function(genFun) {\r\n    if (Object.setPrototypeOf) {\r\n      Object.setPrototypeOf(genFun, GeneratorFunctionPrototype);\r\n    } else {\r\n      genFun.__proto__ = GeneratorFunctionPrototype;\r\n      if (!(toStringTagSymbol in genFun)) {\r\n        genFun[toStringTagSymbol] = \"GeneratorFunction\";\r\n      }\r\n    }\r\n    genFun.prototype = Object.create(Gp);\r\n    return genFun;\r\n  };\r\n\r\n  // Within the body of any async function, `await x` is transformed to\r\n  // `yield regeneratorRuntime.awrap(x)`, so that the runtime can test\r\n  // `hasOwn.call(value, \"__await\")` to determine if the yielded value is\r\n  // meant to be awaited.\r\n  runtime.awrap = function(arg) {\r\n    return { __await: arg };\r\n  };\r\n\r\n  function AsyncIterator(generator) {\r\n    function invoke(method, arg, resolve, reject) {\r\n      var record = tryCatch(generator[method], generator, arg);\r\n      if (record.type === \"throw\") {\r\n        reject(record.arg);\r\n      } else {\r\n        var result = record.arg;\r\n        var value = result.value;\r\n        if (value &&\r\n            typeof value === \"object\" &&\r\n            hasOwn.call(value, \"__await\")) {\r\n          return Promise.resolve(value.__await).then(function(value) {\r\n            invoke(\"next\", value, resolve, reject);\r\n          }, function(err) {\r\n            invoke(\"throw\", err, resolve, reject);\r\n          });\r\n        }\r\n\r\n        return Promise.resolve(value).then(function(unwrapped) {\r\n          // When a yielded Promise is resolved, its final value becomes\r\n          // the .value of the Promise<{value,done}> result for the\r\n          // current iteration. If the Promise is rejected, however, the\r\n          // result for this iteration will be rejected with the same\r\n          // reason. Note that rejections of yielded Promises are not\r\n          // thrown back into the generator function, as is the case\r\n          // when an awaited Promise is rejected. This difference in\r\n          // behavior between yield and await is important, because it\r\n          // allows the consumer to decide what to do with the yielded\r\n          // rejection (swallow it and continue, manually .throw it back\r\n          // into the generator, abandon iteration, whatever). With\r\n          // await, by contrast, there is no opportunity to examine the\r\n          // rejection reason outside the generator function, so the\r\n          // only option is to throw it from the await expression, and\r\n          // let the generator function handle the exception.\r\n          result.value = unwrapped;\r\n          resolve(result);\r\n        }, reject);\r\n      }\r\n    }\r\n\r\n    if (typeof global.process === \"object\" && global.process.domain) {\r\n      invoke = global.process.domain.bind(invoke);\r\n    }\r\n\r\n    var previousPromise;\r\n\r\n    function enqueue(method, arg) {\r\n      function callInvokeWithMethodAndArg() {\r\n        return new Promise(function(resolve, reject) {\r\n          invoke(method, arg, resolve, reject);\r\n        });\r\n      }\r\n\r\n      return previousPromise =\r\n        // If enqueue has been called before, then we want to wait until\r\n        // all previous Promises have been resolved before calling invoke,\r\n        // so that results are always delivered in the correct order. If\r\n        // enqueue has not been called before, then it is important to\r\n        // call invoke immediately, without waiting on a callback to fire,\r\n        // so that the async generator function has the opportunity to do\r\n        // any necessary setup in a predictable way. This predictability\r\n        // is why the Promise constructor synchronously invokes its\r\n        // executor callback, and why async functions synchronously\r\n        // execute code before the first await. Since we implement simple\r\n        // async functions in terms of async generators, it is especially\r\n        // important to get this right, even though it requires care.\r\n        previousPromise ? previousPromise.then(\r\n          callInvokeWithMethodAndArg,\r\n          // Avoid propagating failures to Promises returned by later\r\n          // invocations of the iterator.\r\n          callInvokeWithMethodAndArg\r\n        ) : callInvokeWithMethodAndArg();\r\n    }\r\n\r\n    // Define the unified helper method that is used to implement .next,\r\n    // .throw, and .return (see defineIteratorMethods).\r\n    this._invoke = enqueue;\r\n  }\r\n\r\n  defineIteratorMethods(AsyncIterator.prototype);\r\n  AsyncIterator.prototype[asyncIteratorSymbol] = function () {\r\n    return this;\r\n  };\r\n  runtime.AsyncIterator = AsyncIterator;\r\n\r\n  // Note that simple async functions are implemented on top of\r\n  // AsyncIterator objects; they just return a Promise for the value of\r\n  // the final result produced by the iterator.\r\n  runtime.async = function(innerFn, outerFn, self, tryLocsList) {\r\n    var iter = new AsyncIterator(\r\n      wrap(innerFn, outerFn, self, tryLocsList)\r\n    );\r\n\r\n    return runtime.isGeneratorFunction(outerFn)\r\n      ? iter // If outerFn is a generator, return the full iterator.\r\n      : iter.next().then(function(result) {\r\n          return result.done ? result.value : iter.next();\r\n        });\r\n  };\r\n\r\n  function makeInvokeMethod(innerFn, self, context) {\r\n    var state = GenStateSuspendedStart;\r\n\r\n    return function invoke(method, arg) {\r\n      if (state === GenStateExecuting) {\r\n        throw new Error(\"Generator is already running\");\r\n      }\r\n\r\n      if (state === GenStateCompleted) {\r\n        if (method === \"throw\") {\r\n          throw arg;\r\n        }\r\n\r\n        // Be forgiving, per 25.3.3.3.3 of the spec:\r\n        // https://people.mozilla.org/~jorendorff/es6-draft.html#sec-generatorresume\r\n        return doneResult();\r\n      }\r\n\r\n      context.method = method;\r\n      context.arg = arg;\r\n\r\n      while (true) {\r\n        var delegate = context.delegate;\r\n        if (delegate) {\r\n          var delegateResult = maybeInvokeDelegate(delegate, context);\r\n          if (delegateResult) {\r\n            if (delegateResult === ContinueSentinel) continue;\r\n            return delegateResult;\r\n          }\r\n        }\r\n\r\n        if (context.method === \"next\") {\r\n          // Setting context._sent for legacy support of Babel's\r\n          // function.sent implementation.\r\n          context.sent = context._sent = context.arg;\r\n\r\n        } else if (context.method === \"throw\") {\r\n          if (state === GenStateSuspendedStart) {\r\n            state = GenStateCompleted;\r\n            throw context.arg;\r\n          }\r\n\r\n          context.dispatchException(context.arg);\r\n\r\n        } else if (context.method === \"return\") {\r\n          context.abrupt(\"return\", context.arg);\r\n        }\r\n\r\n        state = GenStateExecuting;\r\n\r\n        var record = tryCatch(innerFn, self, context);\r\n        if (record.type === \"normal\") {\r\n          // If an exception is thrown from innerFn, we leave state ===\r\n          // GenStateExecuting and loop back for another invocation.\r\n          state = context.done\r\n            ? GenStateCompleted\r\n            : GenStateSuspendedYield;\r\n\r\n          if (record.arg === ContinueSentinel) {\r\n            continue;\r\n          }\r\n\r\n          return {\r\n            value: record.arg,\r\n            done: context.done\r\n          };\r\n\r\n        } else if (record.type === \"throw\") {\r\n          state = GenStateCompleted;\r\n          // Dispatch the exception by looping back around to the\r\n          // context.dispatchException(context.arg) call above.\r\n          context.method = \"throw\";\r\n          context.arg = record.arg;\r\n        }\r\n      }\r\n    };\r\n  }\r\n\r\n  // Call delegate.iterator[context.method](context.arg) and handle the\r\n  // result, either by returning a { value, done } result from the\r\n  // delegate iterator, or by modifying context.method and context.arg,\r\n  // setting context.delegate to null, and returning the ContinueSentinel.\r\n  function maybeInvokeDelegate(delegate, context) {\r\n    var method = delegate.iterator[context.method];\r\n    if (method === undefined) {\r\n      // A .throw or .return when the delegate iterator has no .throw\r\n      // method always terminates the yield* loop.\r\n      context.delegate = null;\r\n\r\n      if (context.method === \"throw\") {\r\n        if (delegate.iterator.return) {\r\n          // If the delegate iterator has a return method, give it a\r\n          // chance to clean up.\r\n          context.method = \"return\";\r\n          context.arg = undefined;\r\n          maybeInvokeDelegate(delegate, context);\r\n\r\n          if (context.method === \"throw\") {\r\n            // If maybeInvokeDelegate(context) changed context.method from\r\n            // \"return\" to \"throw\", let that override the TypeError below.\r\n            return ContinueSentinel;\r\n          }\r\n        }\r\n\r\n        context.method = \"throw\";\r\n        context.arg = new TypeError(\r\n          \"The iterator does not provide a 'throw' method\");\r\n      }\r\n\r\n      return ContinueSentinel;\r\n    }\r\n\r\n    var record = tryCatch(method, delegate.iterator, context.arg);\r\n\r\n    if (record.type === \"throw\") {\r\n      context.method = \"throw\";\r\n      context.arg = record.arg;\r\n      context.delegate = null;\r\n      return ContinueSentinel;\r\n    }\r\n\r\n    var info = record.arg;\r\n\r\n    if (! info) {\r\n      context.method = \"throw\";\r\n      context.arg = new TypeError(\"iterator result is not an object\");\r\n      context.delegate = null;\r\n      return ContinueSentinel;\r\n    }\r\n\r\n    if (info.done) {\r\n      // Assign the result of the finished delegate to the temporary\r\n      // variable specified by delegate.resultName (see delegateYield).\r\n      context[delegate.resultName] = info.value;\r\n\r\n      // Resume execution at the desired location (see delegateYield).\r\n      context.next = delegate.nextLoc;\r\n\r\n      // If context.method was \"throw\" but the delegate handled the\r\n      // exception, let the outer generator proceed normally. If\r\n      // context.method was \"next\", forget context.arg since it has been\r\n      // \"consumed\" by the delegate iterator. If context.method was\r\n      // \"return\", allow the original .return call to continue in the\r\n      // outer generator.\r\n      if (context.method !== \"return\") {\r\n        context.method = \"next\";\r\n        context.arg = undefined;\r\n      }\r\n\r\n    } else {\r\n      // Re-yield the result returned by the delegate method.\r\n      return info;\r\n    }\r\n\r\n    // The delegate iterator is finished, so forget it and continue with\r\n    // the outer generator.\r\n    context.delegate = null;\r\n    return ContinueSentinel;\r\n  }\r\n\r\n  // Define Generator.prototype.{next,throw,return} in terms of the\r\n  // unified ._invoke helper method.\r\n  defineIteratorMethods(Gp);\r\n\r\n  Gp[toStringTagSymbol] = \"Generator\";\r\n\r\n  // A Generator should always return itself as the iterator object when the\r\n  // @@iterator function is called on it. Some browsers' implementations of the\r\n  // iterator prototype chain incorrectly implement this, causing the Generator\r\n  // object to not be returned from this call. This ensures that doesn't happen.\r\n  // See https://github.com/facebook/regenerator/issues/274 for more details.\r\n  Gp[iteratorSymbol] = function() {\r\n    return this;\r\n  };\r\n\r\n  Gp.toString = function() {\r\n    return \"[object Generator]\";\r\n  };\r\n\r\n  function pushTryEntry(locs) {\r\n    var entry = { tryLoc: locs[0] };\r\n\r\n    if (1 in locs) {\r\n      entry.catchLoc = locs[1];\r\n    }\r\n\r\n    if (2 in locs) {\r\n      entry.finallyLoc = locs[2];\r\n      entry.afterLoc = locs[3];\r\n    }\r\n\r\n    this.tryEntries.push(entry);\r\n  }\r\n\r\n  function resetTryEntry(entry) {\r\n    var record = entry.completion || {};\r\n    record.type = \"normal\";\r\n    delete record.arg;\r\n    entry.completion = record;\r\n  }\r\n\r\n  function Context(tryLocsList) {\r\n    // The root entry object (effectively a try statement without a catch\r\n    // or a finally block) gives us a place to store values thrown from\r\n    // locations where there is no enclosing try statement.\r\n    this.tryEntries = [{ tryLoc: \"root\" }];\r\n    tryLocsList.forEach(pushTryEntry, this);\r\n    this.reset(true);\r\n  }\r\n\r\n  runtime.keys = function(object) {\r\n    var keys = [];\r\n    for (var key in object) {\r\n      keys.push(key);\r\n    }\r\n    keys.reverse();\r\n\r\n    // Rather than returning an object with a next method, we keep\r\n    // things simple and return the next function itself.\r\n    return function next() {\r\n      while (keys.length) {\r\n        var key = keys.pop();\r\n        if (key in object) {\r\n          next.value = key;\r\n          next.done = false;\r\n          return next;\r\n        }\r\n      }\r\n\r\n      // To avoid creating an additional object, we just hang the .value\r\n      // and .done properties off the next function object itself. This\r\n      // also ensures that the minifier will not anonymize the function.\r\n      next.done = true;\r\n      return next;\r\n    };\r\n  };\r\n\r\n  function values(iterable) {\r\n    if (iterable) {\r\n      var iteratorMethod = iterable[iteratorSymbol];\r\n      if (iteratorMethod) {\r\n        return iteratorMethod.call(iterable);\r\n      }\r\n\r\n      if (typeof iterable.next === \"function\") {\r\n        return iterable;\r\n      }\r\n\r\n      if (!isNaN(iterable.length)) {\r\n        var i = -1, next = function next() {\r\n          while (++i < iterable.length) {\r\n            if (hasOwn.call(iterable, i)) {\r\n              next.value = iterable[i];\r\n              next.done = false;\r\n              return next;\r\n            }\r\n          }\r\n\r\n          next.value = undefined;\r\n          next.done = true;\r\n\r\n          return next;\r\n        };\r\n\r\n        return next.next = next;\r\n      }\r\n    }\r\n\r\n    // Return an iterator with no values.\r\n    return { next: doneResult };\r\n  }\r\n  runtime.values = values;\r\n\r\n  function doneResult() {\r\n    return { value: undefined, done: true };\r\n  }\r\n\r\n  Context.prototype = {\r\n    constructor: Context,\r\n\r\n    reset: function(skipTempReset) {\r\n      this.prev = 0;\r\n      this.next = 0;\r\n      // Resetting context._sent for legacy support of Babel's\r\n      // function.sent implementation.\r\n      this.sent = this._sent = undefined;\r\n      this.done = false;\r\n      this.delegate = null;\r\n\r\n      this.method = \"next\";\r\n      this.arg = undefined;\r\n\r\n      this.tryEntries.forEach(resetTryEntry);\r\n\r\n      if (!skipTempReset) {\r\n        for (var name in this) {\r\n          // Not sure about the optimal order of these conditions:\r\n          if (name.charAt(0) === \"t\" &&\r\n              hasOwn.call(this, name) &&\r\n              !isNaN(+name.slice(1))) {\r\n            this[name] = undefined;\r\n          }\r\n        }\r\n      }\r\n    },\r\n\r\n    stop: function() {\r\n      this.done = true;\r\n\r\n      var rootEntry = this.tryEntries[0];\r\n      var rootRecord = rootEntry.completion;\r\n      if (rootRecord.type === \"throw\") {\r\n        throw rootRecord.arg;\r\n      }\r\n\r\n      return this.rval;\r\n    },\r\n\r\n    dispatchException: function(exception) {\r\n      if (this.done) {\r\n        throw exception;\r\n      }\r\n\r\n      var context = this;\r\n      function handle(loc, caught) {\r\n        record.type = \"throw\";\r\n        record.arg = exception;\r\n        context.next = loc;\r\n\r\n        if (caught) {\r\n          // If the dispatched exception was caught by a catch block,\r\n          // then let that catch block handle the exception normally.\r\n          context.method = \"next\";\r\n          context.arg = undefined;\r\n        }\r\n\r\n        return !! caught;\r\n      }\r\n\r\n      for (var i = this.tryEntries.length - 1; i >= 0; --i) {\r\n        var entry = this.tryEntries[i];\r\n        var record = entry.completion;\r\n\r\n        if (entry.tryLoc === \"root\") {\r\n          // Exception thrown outside of any try block that could handle\r\n          // it, so set the completion value of the entire function to\r\n          // throw the exception.\r\n          return handle(\"end\");\r\n        }\r\n\r\n        if (entry.tryLoc <= this.prev) {\r\n          var hasCatch = hasOwn.call(entry, \"catchLoc\");\r\n          var hasFinally = hasOwn.call(entry, \"finallyLoc\");\r\n\r\n          if (hasCatch && hasFinally) {\r\n            if (this.prev < entry.catchLoc) {\r\n              return handle(entry.catchLoc, true);\r\n            } else if (this.prev < entry.finallyLoc) {\r\n              return handle(entry.finallyLoc);\r\n            }\r\n\r\n          } else if (hasCatch) {\r\n            if (this.prev < entry.catchLoc) {\r\n              return handle(entry.catchLoc, true);\r\n            }\r\n\r\n          } else if (hasFinally) {\r\n            if (this.prev < entry.finallyLoc) {\r\n              return handle(entry.finallyLoc);\r\n            }\r\n\r\n          } else {\r\n            throw new Error(\"try statement without catch or finally\");\r\n          }\r\n        }\r\n      }\r\n    },\r\n\r\n    abrupt: function(type, arg) {\r\n      for (var i = this.tryEntries.length - 1; i >= 0; --i) {\r\n        var entry = this.tryEntries[i];\r\n        if (entry.tryLoc <= this.prev &&\r\n            hasOwn.call(entry, \"finallyLoc\") &&\r\n            this.prev < entry.finallyLoc) {\r\n          var finallyEntry = entry;\r\n          break;\r\n        }\r\n      }\r\n\r\n      if (finallyEntry &&\r\n          (type === \"break\" ||\r\n           type === \"continue\") &&\r\n          finallyEntry.tryLoc <= arg &&\r\n          arg <= finallyEntry.finallyLoc) {\r\n        // Ignore the finally entry if control is not jumping to a\r\n        // location outside the try/catch block.\r\n        finallyEntry = null;\r\n      }\r\n\r\n      var record = finallyEntry ? finallyEntry.completion : {};\r\n      record.type = type;\r\n      record.arg = arg;\r\n\r\n      if (finallyEntry) {\r\n        this.method = \"next\";\r\n        this.next = finallyEntry.finallyLoc;\r\n        return ContinueSentinel;\r\n      }\r\n\r\n      return this.complete(record);\r\n    },\r\n\r\n    complete: function(record, afterLoc) {\r\n      if (record.type === \"throw\") {\r\n        throw record.arg;\r\n      }\r\n\r\n      if (record.type === \"break\" ||\r\n          record.type === \"continue\") {\r\n        this.next = record.arg;\r\n      } else if (record.type === \"return\") {\r\n        this.rval = this.arg = record.arg;\r\n        this.method = \"return\";\r\n        this.next = \"end\";\r\n      } else if (record.type === \"normal\" && afterLoc) {\r\n        this.next = afterLoc;\r\n      }\r\n\r\n      return ContinueSentinel;\r\n    },\r\n\r\n    finish: function(finallyLoc) {\r\n      for (var i = this.tryEntries.length - 1; i >= 0; --i) {\r\n        var entry = this.tryEntries[i];\r\n        if (entry.finallyLoc === finallyLoc) {\r\n          this.complete(entry.completion, entry.afterLoc);\r\n          resetTryEntry(entry);\r\n          return ContinueSentinel;\r\n        }\r\n      }\r\n    },\r\n\r\n    \"catch\": function(tryLoc) {\r\n      for (var i = this.tryEntries.length - 1; i >= 0; --i) {\r\n        var entry = this.tryEntries[i];\r\n        if (entry.tryLoc === tryLoc) {\r\n          var record = entry.completion;\r\n          if (record.type === \"throw\") {\r\n            var thrown = record.arg;\r\n            resetTryEntry(entry);\r\n          }\r\n          return thrown;\r\n        }\r\n      }\r\n\r\n      // The context.catch method must only be called with a location\r\n      // argument that corresponds to a known catch block.\r\n      throw new Error(\"illegal catch attempt\");\r\n    },\r\n\r\n    delegateYield: function(iterable, resultName, nextLoc) {\r\n      this.delegate = {\r\n        iterator: values(iterable),\r\n        resultName: resultName,\r\n        nextLoc: nextLoc\r\n      };\r\n\r\n      if (this.method === \"next\") {\r\n        // Deliberately forget the last sent value so that we don't\r\n        // accidentally pass it on to the delegate.\r\n        this.arg = undefined;\r\n      }\r\n\r\n      return ContinueSentinel;\r\n    }\r\n  };\r\n})(\r\n  // Among the various tricks for obtaining a reference to the global\r\n  // object, this seems to be the most reliable technique that does not\r\n  // use indirect eval (which violates Content Security Policy).\r\n  typeof __webpack_require__.g === \"object\" ? __webpack_require__.g :\r\n  typeof window === \"object\" ? window :\r\n  typeof self === \"object\" ? self : this\r\n);\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/babel-polyfill/node_modules/regenerator-runtime/runtime.js?");

/***/ }),

/***/ "./node_modules/core-js/fn/regexp/escape.js":
/*!**************************************************!*\
  !*** ./node_modules/core-js/fn/regexp/escape.js ***!
  \**************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("__webpack_require__(/*! ../../modules/core.regexp.escape */ \"./node_modules/core-js/modules/core.regexp.escape.js\");\r\nmodule.exports = __webpack_require__(/*! ../../modules/_core */ \"./node_modules/core-js/modules/_core.js\").RegExp.escape;\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/fn/regexp/escape.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_a-function.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_a-function.js ***!
  \*****************************************************/
/***/ (function(module) {

eval("module.exports = function (it) {\r\n  if (typeof it != 'function') throw TypeError(it + ' is not a function!');\r\n  return it;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_a-function.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_a-number-value.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/_a-number-value.js ***!
  \*********************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var cof = __webpack_require__(/*! ./_cof */ \"./node_modules/core-js/modules/_cof.js\");\r\nmodule.exports = function (it, msg) {\r\n  if (typeof it != 'number' && cof(it) != 'Number') throw TypeError(msg);\r\n  return +it;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_a-number-value.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_add-to-unscopables.js":
/*!*************************************************************!*\
  !*** ./node_modules/core-js/modules/_add-to-unscopables.js ***!
  \*************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// 22.1.3.31 Array.prototype[@@unscopables]\r\nvar UNSCOPABLES = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('unscopables');\r\nvar ArrayProto = Array.prototype;\r\nif (ArrayProto[UNSCOPABLES] == undefined) __webpack_require__(/*! ./_hide */ \"./node_modules/core-js/modules/_hide.js\")(ArrayProto, UNSCOPABLES, {});\r\nmodule.exports = function (key) {\r\n  ArrayProto[UNSCOPABLES][key] = true;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_add-to-unscopables.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_advance-string-index.js":
/*!***************************************************************!*\
  !*** ./node_modules/core-js/modules/_advance-string-index.js ***!
  \***************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar at = __webpack_require__(/*! ./_string-at */ \"./node_modules/core-js/modules/_string-at.js\")(true);\r\n\r\n // `AdvanceStringIndex` abstract operation\r\n// https://tc39.github.io/ecma262/#sec-advancestringindex\r\nmodule.exports = function (S, index, unicode) {\r\n  return index + (unicode ? at(S, index).length : 1);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_advance-string-index.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_an-instance.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/_an-instance.js ***!
  \******************************************************/
/***/ (function(module) {

eval("module.exports = function (it, Constructor, name, forbiddenField) {\r\n  if (!(it instanceof Constructor) || (forbiddenField !== undefined && forbiddenField in it)) {\r\n    throw TypeError(name + ': incorrect invocation!');\r\n  } return it;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_an-instance.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_an-object.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/_an-object.js ***!
  \****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nmodule.exports = function (it) {\r\n  if (!isObject(it)) throw TypeError(it + ' is not an object!');\r\n  return it;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_an-object.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_array-copy-within.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/_array-copy-within.js ***!
  \************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("// 22.1.3.3 Array.prototype.copyWithin(target, start, end = this.length)\r\n\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar toAbsoluteIndex = __webpack_require__(/*! ./_to-absolute-index */ \"./node_modules/core-js/modules/_to-absolute-index.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\n\r\nmodule.exports = [].copyWithin || function copyWithin(target /* = 0 */, start /* = 0, end = @length */) {\r\n  var O = toObject(this);\r\n  var len = toLength(O.length);\r\n  var to = toAbsoluteIndex(target, len);\r\n  var from = toAbsoluteIndex(start, len);\r\n  var end = arguments.length > 2 ? arguments[2] : undefined;\r\n  var count = Math.min((end === undefined ? len : toAbsoluteIndex(end, len)) - from, len - to);\r\n  var inc = 1;\r\n  if (from < to && to < from + count) {\r\n    inc = -1;\r\n    from += count - 1;\r\n    to += count - 1;\r\n  }\r\n  while (count-- > 0) {\r\n    if (from in O) O[to] = O[from];\r\n    else delete O[to];\r\n    to += inc;\r\n    from += inc;\r\n  } return O;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_array-copy-within.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_array-fill.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_array-fill.js ***!
  \*****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("// 22.1.3.6 Array.prototype.fill(value, start = 0, end = this.length)\r\n\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar toAbsoluteIndex = __webpack_require__(/*! ./_to-absolute-index */ \"./node_modules/core-js/modules/_to-absolute-index.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nmodule.exports = function fill(value /* , start = 0, end = @length */) {\r\n  var O = toObject(this);\r\n  var length = toLength(O.length);\r\n  var aLen = arguments.length;\r\n  var index = toAbsoluteIndex(aLen > 1 ? arguments[1] : undefined, length);\r\n  var end = aLen > 2 ? arguments[2] : undefined;\r\n  var endPos = end === undefined ? length : toAbsoluteIndex(end, length);\r\n  while (endPos > index) O[index++] = value;\r\n  return O;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_array-fill.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_array-from-iterable.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/_array-from-iterable.js ***!
  \**************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var forOf = __webpack_require__(/*! ./_for-of */ \"./node_modules/core-js/modules/_for-of.js\");\r\n\r\nmodule.exports = function (iter, ITERATOR) {\r\n  var result = [];\r\n  forOf(iter, false, result.push, result, ITERATOR);\r\n  return result;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_array-from-iterable.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_array-includes.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/_array-includes.js ***!
  \*********************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// false -> Array#indexOf\r\n// true  -> Array#includes\r\nvar toIObject = __webpack_require__(/*! ./_to-iobject */ \"./node_modules/core-js/modules/_to-iobject.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nvar toAbsoluteIndex = __webpack_require__(/*! ./_to-absolute-index */ \"./node_modules/core-js/modules/_to-absolute-index.js\");\r\nmodule.exports = function (IS_INCLUDES) {\r\n  return function ($this, el, fromIndex) {\r\n    var O = toIObject($this);\r\n    var length = toLength(O.length);\r\n    var index = toAbsoluteIndex(fromIndex, length);\r\n    var value;\r\n    // Array#includes uses SameValueZero equality algorithm\r\n    // eslint-disable-next-line no-self-compare\r\n    if (IS_INCLUDES && el != el) while (length > index) {\r\n      value = O[index++];\r\n      // eslint-disable-next-line no-self-compare\r\n      if (value != value) return true;\r\n    // Array#indexOf ignores holes, Array#includes - not\r\n    } else for (;length > index; index++) if (IS_INCLUDES || index in O) {\r\n      if (O[index] === el) return IS_INCLUDES || index || 0;\r\n    } return !IS_INCLUDES && -1;\r\n  };\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_array-includes.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_array-methods.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/_array-methods.js ***!
  \********************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// 0 -> Array#forEach\r\n// 1 -> Array#map\r\n// 2 -> Array#filter\r\n// 3 -> Array#some\r\n// 4 -> Array#every\r\n// 5 -> Array#find\r\n// 6 -> Array#findIndex\r\nvar ctx = __webpack_require__(/*! ./_ctx */ \"./node_modules/core-js/modules/_ctx.js\");\r\nvar IObject = __webpack_require__(/*! ./_iobject */ \"./node_modules/core-js/modules/_iobject.js\");\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nvar asc = __webpack_require__(/*! ./_array-species-create */ \"./node_modules/core-js/modules/_array-species-create.js\");\r\nmodule.exports = function (TYPE, $create) {\r\n  var IS_MAP = TYPE == 1;\r\n  var IS_FILTER = TYPE == 2;\r\n  var IS_SOME = TYPE == 3;\r\n  var IS_EVERY = TYPE == 4;\r\n  var IS_FIND_INDEX = TYPE == 6;\r\n  var NO_HOLES = TYPE == 5 || IS_FIND_INDEX;\r\n  var create = $create || asc;\r\n  return function ($this, callbackfn, that) {\r\n    var O = toObject($this);\r\n    var self = IObject(O);\r\n    var f = ctx(callbackfn, that, 3);\r\n    var length = toLength(self.length);\r\n    var index = 0;\r\n    var result = IS_MAP ? create($this, length) : IS_FILTER ? create($this, 0) : undefined;\r\n    var val, res;\r\n    for (;length > index; index++) if (NO_HOLES || index in self) {\r\n      val = self[index];\r\n      res = f(val, index, O);\r\n      if (TYPE) {\r\n        if (IS_MAP) result[index] = res;   // map\r\n        else if (res) switch (TYPE) {\r\n          case 3: return true;             // some\r\n          case 5: return val;              // find\r\n          case 6: return index;            // findIndex\r\n          case 2: result.push(val);        // filter\r\n        } else if (IS_EVERY) return false; // every\r\n      }\r\n    }\r\n    return IS_FIND_INDEX ? -1 : IS_SOME || IS_EVERY ? IS_EVERY : result;\r\n  };\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_array-methods.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_array-reduce.js":
/*!*******************************************************!*\
  !*** ./node_modules/core-js/modules/_array-reduce.js ***!
  \*******************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var aFunction = __webpack_require__(/*! ./_a-function */ \"./node_modules/core-js/modules/_a-function.js\");\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar IObject = __webpack_require__(/*! ./_iobject */ \"./node_modules/core-js/modules/_iobject.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\n\r\nmodule.exports = function (that, callbackfn, aLen, memo, isRight) {\r\n  aFunction(callbackfn);\r\n  var O = toObject(that);\r\n  var self = IObject(O);\r\n  var length = toLength(O.length);\r\n  var index = isRight ? length - 1 : 0;\r\n  var i = isRight ? -1 : 1;\r\n  if (aLen < 2) for (;;) {\r\n    if (index in self) {\r\n      memo = self[index];\r\n      index += i;\r\n      break;\r\n    }\r\n    index += i;\r\n    if (isRight ? index < 0 : length <= index) {\r\n      throw TypeError('Reduce of empty array with no initial value');\r\n    }\r\n  }\r\n  for (;isRight ? index >= 0 : length > index; index += i) if (index in self) {\r\n    memo = callbackfn(memo, self[index], index, O);\r\n  }\r\n  return memo;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_array-reduce.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_array-species-constructor.js":
/*!********************************************************************!*\
  !*** ./node_modules/core-js/modules/_array-species-constructor.js ***!
  \********************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar isArray = __webpack_require__(/*! ./_is-array */ \"./node_modules/core-js/modules/_is-array.js\");\r\nvar SPECIES = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('species');\r\n\r\nmodule.exports = function (original) {\r\n  var C;\r\n  if (isArray(original)) {\r\n    C = original.constructor;\r\n    // cross-realm fallback\r\n    if (typeof C == 'function' && (C === Array || isArray(C.prototype))) C = undefined;\r\n    if (isObject(C)) {\r\n      C = C[SPECIES];\r\n      if (C === null) C = undefined;\r\n    }\r\n  } return C === undefined ? Array : C;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_array-species-constructor.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_array-species-create.js":
/*!***************************************************************!*\
  !*** ./node_modules/core-js/modules/_array-species-create.js ***!
  \***************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// 9.4.2.3 ArraySpeciesCreate(originalArray, length)\r\nvar speciesConstructor = __webpack_require__(/*! ./_array-species-constructor */ \"./node_modules/core-js/modules/_array-species-constructor.js\");\r\n\r\nmodule.exports = function (original, length) {\r\n  return new (speciesConstructor(original))(length);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_array-species-create.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_bind.js":
/*!***********************************************!*\
  !*** ./node_modules/core-js/modules/_bind.js ***!
  \***********************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar aFunction = __webpack_require__(/*! ./_a-function */ \"./node_modules/core-js/modules/_a-function.js\");\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar invoke = __webpack_require__(/*! ./_invoke */ \"./node_modules/core-js/modules/_invoke.js\");\r\nvar arraySlice = [].slice;\r\nvar factories = {};\r\n\r\nvar construct = function (F, len, args) {\r\n  if (!(len in factories)) {\r\n    for (var n = [], i = 0; i < len; i++) n[i] = 'a[' + i + ']';\r\n    // eslint-disable-next-line no-new-func\r\n    factories[len] = Function('F,a', 'return new F(' + n.join(',') + ')');\r\n  } return factories[len](F, args);\r\n};\r\n\r\nmodule.exports = Function.bind || function bind(that /* , ...args */) {\r\n  var fn = aFunction(this);\r\n  var partArgs = arraySlice.call(arguments, 1);\r\n  var bound = function (/* args... */) {\r\n    var args = partArgs.concat(arraySlice.call(arguments));\r\n    return this instanceof bound ? construct(fn, args.length, args) : invoke(fn, args, that);\r\n  };\r\n  if (isObject(fn.prototype)) bound.prototype = fn.prototype;\r\n  return bound;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_bind.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_classof.js":
/*!**************************************************!*\
  !*** ./node_modules/core-js/modules/_classof.js ***!
  \**************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// getting tag from 19.1.3.6 Object.prototype.toString()\r\nvar cof = __webpack_require__(/*! ./_cof */ \"./node_modules/core-js/modules/_cof.js\");\r\nvar TAG = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('toStringTag');\r\n// ES3 wrong here\r\nvar ARG = cof(function () { return arguments; }()) == 'Arguments';\r\n\r\n// fallback for IE11 Script Access Denied error\r\nvar tryGet = function (it, key) {\r\n  try {\r\n    return it[key];\r\n  } catch (e) { /* empty */ }\r\n};\r\n\r\nmodule.exports = function (it) {\r\n  var O, T, B;\r\n  return it === undefined ? 'Undefined' : it === null ? 'Null'\r\n    // @@toStringTag case\r\n    : typeof (T = tryGet(O = Object(it), TAG)) == 'string' ? T\r\n    // builtinTag case\r\n    : ARG ? cof(O)\r\n    // ES3 arguments fallback\r\n    : (B = cof(O)) == 'Object' && typeof O.callee == 'function' ? 'Arguments' : B;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_classof.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_cof.js":
/*!**********************************************!*\
  !*** ./node_modules/core-js/modules/_cof.js ***!
  \**********************************************/
/***/ (function(module) {

eval("var toString = {}.toString;\r\n\r\nmodule.exports = function (it) {\r\n  return toString.call(it).slice(8, -1);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_cof.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_collection-strong.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/_collection-strong.js ***!
  \************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar dP = __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\").f;\r\nvar create = __webpack_require__(/*! ./_object-create */ \"./node_modules/core-js/modules/_object-create.js\");\r\nvar redefineAll = __webpack_require__(/*! ./_redefine-all */ \"./node_modules/core-js/modules/_redefine-all.js\");\r\nvar ctx = __webpack_require__(/*! ./_ctx */ \"./node_modules/core-js/modules/_ctx.js\");\r\nvar anInstance = __webpack_require__(/*! ./_an-instance */ \"./node_modules/core-js/modules/_an-instance.js\");\r\nvar forOf = __webpack_require__(/*! ./_for-of */ \"./node_modules/core-js/modules/_for-of.js\");\r\nvar $iterDefine = __webpack_require__(/*! ./_iter-define */ \"./node_modules/core-js/modules/_iter-define.js\");\r\nvar step = __webpack_require__(/*! ./_iter-step */ \"./node_modules/core-js/modules/_iter-step.js\");\r\nvar setSpecies = __webpack_require__(/*! ./_set-species */ \"./node_modules/core-js/modules/_set-species.js\");\r\nvar DESCRIPTORS = __webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\");\r\nvar fastKey = __webpack_require__(/*! ./_meta */ \"./node_modules/core-js/modules/_meta.js\").fastKey;\r\nvar validate = __webpack_require__(/*! ./_validate-collection */ \"./node_modules/core-js/modules/_validate-collection.js\");\r\nvar SIZE = DESCRIPTORS ? '_s' : 'size';\r\n\r\nvar getEntry = function (that, key) {\r\n  // fast case\r\n  var index = fastKey(key);\r\n  var entry;\r\n  if (index !== 'F') return that._i[index];\r\n  // frozen object case\r\n  for (entry = that._f; entry; entry = entry.n) {\r\n    if (entry.k == key) return entry;\r\n  }\r\n};\r\n\r\nmodule.exports = {\r\n  getConstructor: function (wrapper, NAME, IS_MAP, ADDER) {\r\n    var C = wrapper(function (that, iterable) {\r\n      anInstance(that, C, NAME, '_i');\r\n      that._t = NAME;         // collection type\r\n      that._i = create(null); // index\r\n      that._f = undefined;    // first entry\r\n      that._l = undefined;    // last entry\r\n      that[SIZE] = 0;         // size\r\n      if (iterable != undefined) forOf(iterable, IS_MAP, that[ADDER], that);\r\n    });\r\n    redefineAll(C.prototype, {\r\n      // 23.1.3.1 Map.prototype.clear()\r\n      // 23.2.3.2 Set.prototype.clear()\r\n      clear: function clear() {\r\n        for (var that = validate(this, NAME), data = that._i, entry = that._f; entry; entry = entry.n) {\r\n          entry.r = true;\r\n          if (entry.p) entry.p = entry.p.n = undefined;\r\n          delete data[entry.i];\r\n        }\r\n        that._f = that._l = undefined;\r\n        that[SIZE] = 0;\r\n      },\r\n      // 23.1.3.3 Map.prototype.delete(key)\r\n      // 23.2.3.4 Set.prototype.delete(value)\r\n      'delete': function (key) {\r\n        var that = validate(this, NAME);\r\n        var entry = getEntry(that, key);\r\n        if (entry) {\r\n          var next = entry.n;\r\n          var prev = entry.p;\r\n          delete that._i[entry.i];\r\n          entry.r = true;\r\n          if (prev) prev.n = next;\r\n          if (next) next.p = prev;\r\n          if (that._f == entry) that._f = next;\r\n          if (that._l == entry) that._l = prev;\r\n          that[SIZE]--;\r\n        } return !!entry;\r\n      },\r\n      // 23.2.3.6 Set.prototype.forEach(callbackfn, thisArg = undefined)\r\n      // 23.1.3.5 Map.prototype.forEach(callbackfn, thisArg = undefined)\r\n      forEach: function forEach(callbackfn /* , that = undefined */) {\r\n        validate(this, NAME);\r\n        var f = ctx(callbackfn, arguments.length > 1 ? arguments[1] : undefined, 3);\r\n        var entry;\r\n        while (entry = entry ? entry.n : this._f) {\r\n          f(entry.v, entry.k, this);\r\n          // revert to the last existing entry\r\n          while (entry && entry.r) entry = entry.p;\r\n        }\r\n      },\r\n      // 23.1.3.7 Map.prototype.has(key)\r\n      // 23.2.3.7 Set.prototype.has(value)\r\n      has: function has(key) {\r\n        return !!getEntry(validate(this, NAME), key);\r\n      }\r\n    });\r\n    if (DESCRIPTORS) dP(C.prototype, 'size', {\r\n      get: function () {\r\n        return validate(this, NAME)[SIZE];\r\n      }\r\n    });\r\n    return C;\r\n  },\r\n  def: function (that, key, value) {\r\n    var entry = getEntry(that, key);\r\n    var prev, index;\r\n    // change existing entry\r\n    if (entry) {\r\n      entry.v = value;\r\n    // create new entry\r\n    } else {\r\n      that._l = entry = {\r\n        i: index = fastKey(key, true), // <- index\r\n        k: key,                        // <- key\r\n        v: value,                      // <- value\r\n        p: prev = that._l,             // <- previous entry\r\n        n: undefined,                  // <- next entry\r\n        r: false                       // <- removed\r\n      };\r\n      if (!that._f) that._f = entry;\r\n      if (prev) prev.n = entry;\r\n      that[SIZE]++;\r\n      // add to index\r\n      if (index !== 'F') that._i[index] = entry;\r\n    } return that;\r\n  },\r\n  getEntry: getEntry,\r\n  setStrong: function (C, NAME, IS_MAP) {\r\n    // add .keys, .values, .entries, [@@iterator]\r\n    // 23.1.3.4, 23.1.3.8, 23.1.3.11, 23.1.3.12, 23.2.3.5, 23.2.3.8, 23.2.3.10, 23.2.3.11\r\n    $iterDefine(C, NAME, function (iterated, kind) {\r\n      this._t = validate(iterated, NAME); // target\r\n      this._k = kind;                     // kind\r\n      this._l = undefined;                // previous\r\n    }, function () {\r\n      var that = this;\r\n      var kind = that._k;\r\n      var entry = that._l;\r\n      // revert to the last existing entry\r\n      while (entry && entry.r) entry = entry.p;\r\n      // get next entry\r\n      if (!that._t || !(that._l = entry = entry ? entry.n : that._t._f)) {\r\n        // or finish the iteration\r\n        that._t = undefined;\r\n        return step(1);\r\n      }\r\n      // return step by kind\r\n      if (kind == 'keys') return step(0, entry.k);\r\n      if (kind == 'values') return step(0, entry.v);\r\n      return step(0, [entry.k, entry.v]);\r\n    }, IS_MAP ? 'entries' : 'values', !IS_MAP, true);\r\n\r\n    // add [@@species], 23.1.2.2, 23.2.2.2\r\n    setSpecies(NAME);\r\n  }\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_collection-strong.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_collection-to-json.js":
/*!*************************************************************!*\
  !*** ./node_modules/core-js/modules/_collection-to-json.js ***!
  \*************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// https://github.com/DavidBruant/Map-Set.prototype.toJSON\r\nvar classof = __webpack_require__(/*! ./_classof */ \"./node_modules/core-js/modules/_classof.js\");\r\nvar from = __webpack_require__(/*! ./_array-from-iterable */ \"./node_modules/core-js/modules/_array-from-iterable.js\");\r\nmodule.exports = function (NAME) {\r\n  return function toJSON() {\r\n    if (classof(this) != NAME) throw TypeError(NAME + \"#toJSON isn't generic\");\r\n    return from(this);\r\n  };\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_collection-to-json.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_collection-weak.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/_collection-weak.js ***!
  \**********************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar redefineAll = __webpack_require__(/*! ./_redefine-all */ \"./node_modules/core-js/modules/_redefine-all.js\");\r\nvar getWeak = __webpack_require__(/*! ./_meta */ \"./node_modules/core-js/modules/_meta.js\").getWeak;\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar anInstance = __webpack_require__(/*! ./_an-instance */ \"./node_modules/core-js/modules/_an-instance.js\");\r\nvar forOf = __webpack_require__(/*! ./_for-of */ \"./node_modules/core-js/modules/_for-of.js\");\r\nvar createArrayMethod = __webpack_require__(/*! ./_array-methods */ \"./node_modules/core-js/modules/_array-methods.js\");\r\nvar $has = __webpack_require__(/*! ./_has */ \"./node_modules/core-js/modules/_has.js\");\r\nvar validate = __webpack_require__(/*! ./_validate-collection */ \"./node_modules/core-js/modules/_validate-collection.js\");\r\nvar arrayFind = createArrayMethod(5);\r\nvar arrayFindIndex = createArrayMethod(6);\r\nvar id = 0;\r\n\r\n// fallback for uncaught frozen keys\r\nvar uncaughtFrozenStore = function (that) {\r\n  return that._l || (that._l = new UncaughtFrozenStore());\r\n};\r\nvar UncaughtFrozenStore = function () {\r\n  this.a = [];\r\n};\r\nvar findUncaughtFrozen = function (store, key) {\r\n  return arrayFind(store.a, function (it) {\r\n    return it[0] === key;\r\n  });\r\n};\r\nUncaughtFrozenStore.prototype = {\r\n  get: function (key) {\r\n    var entry = findUncaughtFrozen(this, key);\r\n    if (entry) return entry[1];\r\n  },\r\n  has: function (key) {\r\n    return !!findUncaughtFrozen(this, key);\r\n  },\r\n  set: function (key, value) {\r\n    var entry = findUncaughtFrozen(this, key);\r\n    if (entry) entry[1] = value;\r\n    else this.a.push([key, value]);\r\n  },\r\n  'delete': function (key) {\r\n    var index = arrayFindIndex(this.a, function (it) {\r\n      return it[0] === key;\r\n    });\r\n    if (~index) this.a.splice(index, 1);\r\n    return !!~index;\r\n  }\r\n};\r\n\r\nmodule.exports = {\r\n  getConstructor: function (wrapper, NAME, IS_MAP, ADDER) {\r\n    var C = wrapper(function (that, iterable) {\r\n      anInstance(that, C, NAME, '_i');\r\n      that._t = NAME;      // collection type\r\n      that._i = id++;      // collection id\r\n      that._l = undefined; // leak store for uncaught frozen objects\r\n      if (iterable != undefined) forOf(iterable, IS_MAP, that[ADDER], that);\r\n    });\r\n    redefineAll(C.prototype, {\r\n      // 23.3.3.2 WeakMap.prototype.delete(key)\r\n      // 23.4.3.3 WeakSet.prototype.delete(value)\r\n      'delete': function (key) {\r\n        if (!isObject(key)) return false;\r\n        var data = getWeak(key);\r\n        if (data === true) return uncaughtFrozenStore(validate(this, NAME))['delete'](key);\r\n        return data && $has(data, this._i) && delete data[this._i];\r\n      },\r\n      // 23.3.3.4 WeakMap.prototype.has(key)\r\n      // 23.4.3.4 WeakSet.prototype.has(value)\r\n      has: function has(key) {\r\n        if (!isObject(key)) return false;\r\n        var data = getWeak(key);\r\n        if (data === true) return uncaughtFrozenStore(validate(this, NAME)).has(key);\r\n        return data && $has(data, this._i);\r\n      }\r\n    });\r\n    return C;\r\n  },\r\n  def: function (that, key, value) {\r\n    var data = getWeak(anObject(key), true);\r\n    if (data === true) uncaughtFrozenStore(that).set(key, value);\r\n    else data[that._i] = value;\r\n    return that;\r\n  },\r\n  ufstore: uncaughtFrozenStore\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_collection-weak.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_collection.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_collection.js ***!
  \*****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar redefine = __webpack_require__(/*! ./_redefine */ \"./node_modules/core-js/modules/_redefine.js\");\r\nvar redefineAll = __webpack_require__(/*! ./_redefine-all */ \"./node_modules/core-js/modules/_redefine-all.js\");\r\nvar meta = __webpack_require__(/*! ./_meta */ \"./node_modules/core-js/modules/_meta.js\");\r\nvar forOf = __webpack_require__(/*! ./_for-of */ \"./node_modules/core-js/modules/_for-of.js\");\r\nvar anInstance = __webpack_require__(/*! ./_an-instance */ \"./node_modules/core-js/modules/_an-instance.js\");\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar fails = __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\");\r\nvar $iterDetect = __webpack_require__(/*! ./_iter-detect */ \"./node_modules/core-js/modules/_iter-detect.js\");\r\nvar setToStringTag = __webpack_require__(/*! ./_set-to-string-tag */ \"./node_modules/core-js/modules/_set-to-string-tag.js\");\r\nvar inheritIfRequired = __webpack_require__(/*! ./_inherit-if-required */ \"./node_modules/core-js/modules/_inherit-if-required.js\");\r\n\r\nmodule.exports = function (NAME, wrapper, methods, common, IS_MAP, IS_WEAK) {\r\n  var Base = global[NAME];\r\n  var C = Base;\r\n  var ADDER = IS_MAP ? 'set' : 'add';\r\n  var proto = C && C.prototype;\r\n  var O = {};\r\n  var fixMethod = function (KEY) {\r\n    var fn = proto[KEY];\r\n    redefine(proto, KEY,\r\n      KEY == 'delete' ? function (a) {\r\n        return IS_WEAK && !isObject(a) ? false : fn.call(this, a === 0 ? 0 : a);\r\n      } : KEY == 'has' ? function has(a) {\r\n        return IS_WEAK && !isObject(a) ? false : fn.call(this, a === 0 ? 0 : a);\r\n      } : KEY == 'get' ? function get(a) {\r\n        return IS_WEAK && !isObject(a) ? undefined : fn.call(this, a === 0 ? 0 : a);\r\n      } : KEY == 'add' ? function add(a) { fn.call(this, a === 0 ? 0 : a); return this; }\r\n        : function set(a, b) { fn.call(this, a === 0 ? 0 : a, b); return this; }\r\n    );\r\n  };\r\n  if (typeof C != 'function' || !(IS_WEAK || proto.forEach && !fails(function () {\r\n    new C().entries().next();\r\n  }))) {\r\n    // create collection constructor\r\n    C = common.getConstructor(wrapper, NAME, IS_MAP, ADDER);\r\n    redefineAll(C.prototype, methods);\r\n    meta.NEED = true;\r\n  } else {\r\n    var instance = new C();\r\n    // early implementations not supports chaining\r\n    var HASNT_CHAINING = instance[ADDER](IS_WEAK ? {} : -0, 1) != instance;\r\n    // V8 ~  Chromium 40- weak-collections throws on primitives, but should return false\r\n    var THROWS_ON_PRIMITIVES = fails(function () { instance.has(1); });\r\n    // most early implementations doesn't supports iterables, most modern - not close it correctly\r\n    var ACCEPT_ITERABLES = $iterDetect(function (iter) { new C(iter); }); // eslint-disable-line no-new\r\n    // for early implementations -0 and +0 not the same\r\n    var BUGGY_ZERO = !IS_WEAK && fails(function () {\r\n      // V8 ~ Chromium 42- fails only with 5+ elements\r\n      var $instance = new C();\r\n      var index = 5;\r\n      while (index--) $instance[ADDER](index, index);\r\n      return !$instance.has(-0);\r\n    });\r\n    if (!ACCEPT_ITERABLES) {\r\n      C = wrapper(function (target, iterable) {\r\n        anInstance(target, C, NAME);\r\n        var that = inheritIfRequired(new Base(), target, C);\r\n        if (iterable != undefined) forOf(iterable, IS_MAP, that[ADDER], that);\r\n        return that;\r\n      });\r\n      C.prototype = proto;\r\n      proto.constructor = C;\r\n    }\r\n    if (THROWS_ON_PRIMITIVES || BUGGY_ZERO) {\r\n      fixMethod('delete');\r\n      fixMethod('has');\r\n      IS_MAP && fixMethod('get');\r\n    }\r\n    if (BUGGY_ZERO || HASNT_CHAINING) fixMethod(ADDER);\r\n    // weak collections should not contains .clear method\r\n    if (IS_WEAK && proto.clear) delete proto.clear;\r\n  }\r\n\r\n  setToStringTag(C, NAME);\r\n\r\n  O[NAME] = C;\r\n  $export($export.G + $export.W + $export.F * (C != Base), O);\r\n\r\n  if (!IS_WEAK) common.setStrong(C, NAME, IS_MAP);\r\n\r\n  return C;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_collection.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_core.js":
/*!***********************************************!*\
  !*** ./node_modules/core-js/modules/_core.js ***!
  \***********************************************/
/***/ (function(module) {

eval("var core = module.exports = { version: '2.6.12' };\r\nif (typeof __e == 'number') __e = core; // eslint-disable-line no-undef\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_core.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_create-property.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/_create-property.js ***!
  \**********************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $defineProperty = __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\");\r\nvar createDesc = __webpack_require__(/*! ./_property-desc */ \"./node_modules/core-js/modules/_property-desc.js\");\r\n\r\nmodule.exports = function (object, index, value) {\r\n  if (index in object) $defineProperty.f(object, index, createDesc(0, value));\r\n  else object[index] = value;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_create-property.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_ctx.js":
/*!**********************************************!*\
  !*** ./node_modules/core-js/modules/_ctx.js ***!
  \**********************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// optional / simple context binding\r\nvar aFunction = __webpack_require__(/*! ./_a-function */ \"./node_modules/core-js/modules/_a-function.js\");\r\nmodule.exports = function (fn, that, length) {\r\n  aFunction(fn);\r\n  if (that === undefined) return fn;\r\n  switch (length) {\r\n    case 1: return function (a) {\r\n      return fn.call(that, a);\r\n    };\r\n    case 2: return function (a, b) {\r\n      return fn.call(that, a, b);\r\n    };\r\n    case 3: return function (a, b, c) {\r\n      return fn.call(that, a, b, c);\r\n    };\r\n  }\r\n  return function (/* ...args */) {\r\n    return fn.apply(that, arguments);\r\n  };\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_ctx.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_date-to-iso-string.js":
/*!*************************************************************!*\
  !*** ./node_modules/core-js/modules/_date-to-iso-string.js ***!
  \*************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// 20.3.4.36 / 15.9.5.43 Date.prototype.toISOString()\r\nvar fails = __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\");\r\nvar getTime = Date.prototype.getTime;\r\nvar $toISOString = Date.prototype.toISOString;\r\n\r\nvar lz = function (num) {\r\n  return num > 9 ? num : '0' + num;\r\n};\r\n\r\n// PhantomJS / old WebKit has a broken implementations\r\nmodule.exports = (fails(function () {\r\n  return $toISOString.call(new Date(-5e13 - 1)) != '0385-07-25T07:06:39.999Z';\r\n}) || !fails(function () {\r\n  $toISOString.call(new Date(NaN));\r\n})) ? function toISOString() {\r\n  if (!isFinite(getTime.call(this))) throw RangeError('Invalid time value');\r\n  var d = this;\r\n  var y = d.getUTCFullYear();\r\n  var m = d.getUTCMilliseconds();\r\n  var s = y < 0 ? '-' : y > 9999 ? '+' : '';\r\n  return s + ('00000' + Math.abs(y)).slice(s ? -6 : -4) +\r\n    '-' + lz(d.getUTCMonth() + 1) + '-' + lz(d.getUTCDate()) +\r\n    'T' + lz(d.getUTCHours()) + ':' + lz(d.getUTCMinutes()) +\r\n    ':' + lz(d.getUTCSeconds()) + '.' + (m > 99 ? m : '0' + lz(m)) + 'Z';\r\n} : $toISOString;\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_date-to-iso-string.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_date-to-primitive.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/_date-to-primitive.js ***!
  \************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar toPrimitive = __webpack_require__(/*! ./_to-primitive */ \"./node_modules/core-js/modules/_to-primitive.js\");\r\nvar NUMBER = 'number';\r\n\r\nmodule.exports = function (hint) {\r\n  if (hint !== 'string' && hint !== NUMBER && hint !== 'default') throw TypeError('Incorrect hint');\r\n  return toPrimitive(anObject(this), hint != NUMBER);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_date-to-primitive.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_defined.js":
/*!**************************************************!*\
  !*** ./node_modules/core-js/modules/_defined.js ***!
  \**************************************************/
/***/ (function(module) {

eval("// 7.2.1 RequireObjectCoercible(argument)\r\nmodule.exports = function (it) {\r\n  if (it == undefined) throw TypeError(\"Can't call method on  \" + it);\r\n  return it;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_defined.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_descriptors.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/_descriptors.js ***!
  \******************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// Thank's IE8 for his funny defineProperty\r\nmodule.exports = !__webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\")(function () {\r\n  return Object.defineProperty({}, 'a', { get: function () { return 7; } }).a != 7;\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_descriptors.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_dom-create.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_dom-create.js ***!
  \*****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar document = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\").document;\r\n// typeof document.createElement is 'object' in old IE\r\nvar is = isObject(document) && isObject(document.createElement);\r\nmodule.exports = function (it) {\r\n  return is ? document.createElement(it) : {};\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_dom-create.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_enum-bug-keys.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/_enum-bug-keys.js ***!
  \********************************************************/
/***/ (function(module) {

eval("// IE 8- don't enum bug keys\r\nmodule.exports = (\r\n  'constructor,hasOwnProperty,isPrototypeOf,propertyIsEnumerable,toLocaleString,toString,valueOf'\r\n).split(',');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_enum-bug-keys.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_enum-keys.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/_enum-keys.js ***!
  \****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// all enumerable object keys, includes symbols\r\nvar getKeys = __webpack_require__(/*! ./_object-keys */ \"./node_modules/core-js/modules/_object-keys.js\");\r\nvar gOPS = __webpack_require__(/*! ./_object-gops */ \"./node_modules/core-js/modules/_object-gops.js\");\r\nvar pIE = __webpack_require__(/*! ./_object-pie */ \"./node_modules/core-js/modules/_object-pie.js\");\r\nmodule.exports = function (it) {\r\n  var result = getKeys(it);\r\n  var getSymbols = gOPS.f;\r\n  if (getSymbols) {\r\n    var symbols = getSymbols(it);\r\n    var isEnum = pIE.f;\r\n    var i = 0;\r\n    var key;\r\n    while (symbols.length > i) if (isEnum.call(it, key = symbols[i++])) result.push(key);\r\n  } return result;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_enum-keys.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_export.js":
/*!*************************************************!*\
  !*** ./node_modules/core-js/modules/_export.js ***!
  \*************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar core = __webpack_require__(/*! ./_core */ \"./node_modules/core-js/modules/_core.js\");\r\nvar hide = __webpack_require__(/*! ./_hide */ \"./node_modules/core-js/modules/_hide.js\");\r\nvar redefine = __webpack_require__(/*! ./_redefine */ \"./node_modules/core-js/modules/_redefine.js\");\r\nvar ctx = __webpack_require__(/*! ./_ctx */ \"./node_modules/core-js/modules/_ctx.js\");\r\nvar PROTOTYPE = 'prototype';\r\n\r\nvar $export = function (type, name, source) {\r\n  var IS_FORCED = type & $export.F;\r\n  var IS_GLOBAL = type & $export.G;\r\n  var IS_STATIC = type & $export.S;\r\n  var IS_PROTO = type & $export.P;\r\n  var IS_BIND = type & $export.B;\r\n  var target = IS_GLOBAL ? global : IS_STATIC ? global[name] || (global[name] = {}) : (global[name] || {})[PROTOTYPE];\r\n  var exports = IS_GLOBAL ? core : core[name] || (core[name] = {});\r\n  var expProto = exports[PROTOTYPE] || (exports[PROTOTYPE] = {});\r\n  var key, own, out, exp;\r\n  if (IS_GLOBAL) source = name;\r\n  for (key in source) {\r\n    // contains in native\r\n    own = !IS_FORCED && target && target[key] !== undefined;\r\n    // export native or passed\r\n    out = (own ? target : source)[key];\r\n    // bind timers to global for call from export context\r\n    exp = IS_BIND && own ? ctx(out, global) : IS_PROTO && typeof out == 'function' ? ctx(Function.call, out) : out;\r\n    // extend global\r\n    if (target) redefine(target, key, out, type & $export.U);\r\n    // export\r\n    if (exports[key] != out) hide(exports, key, exp);\r\n    if (IS_PROTO && expProto[key] != out) expProto[key] = out;\r\n  }\r\n};\r\nglobal.core = core;\r\n// type bitmap\r\n$export.F = 1;   // forced\r\n$export.G = 2;   // global\r\n$export.S = 4;   // static\r\n$export.P = 8;   // proto\r\n$export.B = 16;  // bind\r\n$export.W = 32;  // wrap\r\n$export.U = 64;  // safe\r\n$export.R = 128; // real proto method for `library`\r\nmodule.exports = $export;\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_export.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_fails-is-regexp.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/_fails-is-regexp.js ***!
  \**********************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var MATCH = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('match');\r\nmodule.exports = function (KEY) {\r\n  var re = /./;\r\n  try {\r\n    '/./'[KEY](re);\r\n  } catch (e) {\r\n    try {\r\n      re[MATCH] = false;\r\n      return !'/./'[KEY](re);\r\n    } catch (f) { /* empty */ }\r\n  } return true;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_fails-is-regexp.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_fails.js":
/*!************************************************!*\
  !*** ./node_modules/core-js/modules/_fails.js ***!
  \************************************************/
/***/ (function(module) {

eval("module.exports = function (exec) {\r\n  try {\r\n    return !!exec();\r\n  } catch (e) {\r\n    return true;\r\n  }\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_fails.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_fix-re-wks.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_fix-re-wks.js ***!
  \*****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n__webpack_require__(/*! ./es6.regexp.exec */ \"./node_modules/core-js/modules/es6.regexp.exec.js\");\r\nvar redefine = __webpack_require__(/*! ./_redefine */ \"./node_modules/core-js/modules/_redefine.js\");\r\nvar hide = __webpack_require__(/*! ./_hide */ \"./node_modules/core-js/modules/_hide.js\");\r\nvar fails = __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\");\r\nvar defined = __webpack_require__(/*! ./_defined */ \"./node_modules/core-js/modules/_defined.js\");\r\nvar wks = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\");\r\nvar regexpExec = __webpack_require__(/*! ./_regexp-exec */ \"./node_modules/core-js/modules/_regexp-exec.js\");\r\n\r\nvar SPECIES = wks('species');\r\n\r\nvar REPLACE_SUPPORTS_NAMED_GROUPS = !fails(function () {\r\n  // #replace needs built-in support for named groups.\r\n  // #match works fine because it just return the exec results, even if it has\r\n  // a \"grops\" property.\r\n  var re = /./;\r\n  re.exec = function () {\r\n    var result = [];\r\n    result.groups = { a: '7' };\r\n    return result;\r\n  };\r\n  return ''.replace(re, '$<a>') !== '7';\r\n});\r\n\r\nvar SPLIT_WORKS_WITH_OVERWRITTEN_EXEC = (function () {\r\n  // Chrome 51 has a buggy \"split\" implementation when RegExp#exec !== nativeExec\r\n  var re = /(?:)/;\r\n  var originalExec = re.exec;\r\n  re.exec = function () { return originalExec.apply(this, arguments); };\r\n  var result = 'ab'.split(re);\r\n  return result.length === 2 && result[0] === 'a' && result[1] === 'b';\r\n})();\r\n\r\nmodule.exports = function (KEY, length, exec) {\r\n  var SYMBOL = wks(KEY);\r\n\r\n  var DELEGATES_TO_SYMBOL = !fails(function () {\r\n    // String methods call symbol-named RegEp methods\r\n    var O = {};\r\n    O[SYMBOL] = function () { return 7; };\r\n    return ''[KEY](O) != 7;\r\n  });\r\n\r\n  var DELEGATES_TO_EXEC = DELEGATES_TO_SYMBOL ? !fails(function () {\r\n    // Symbol-named RegExp methods call .exec\r\n    var execCalled = false;\r\n    var re = /a/;\r\n    re.exec = function () { execCalled = true; return null; };\r\n    if (KEY === 'split') {\r\n      // RegExp[@@split] doesn't call the regex's exec method, but first creates\r\n      // a new one. We need to return the patched regex when creating the new one.\r\n      re.constructor = {};\r\n      re.constructor[SPECIES] = function () { return re; };\r\n    }\r\n    re[SYMBOL]('');\r\n    return !execCalled;\r\n  }) : undefined;\r\n\r\n  if (\r\n    !DELEGATES_TO_SYMBOL ||\r\n    !DELEGATES_TO_EXEC ||\r\n    (KEY === 'replace' && !REPLACE_SUPPORTS_NAMED_GROUPS) ||\r\n    (KEY === 'split' && !SPLIT_WORKS_WITH_OVERWRITTEN_EXEC)\r\n  ) {\r\n    var nativeRegExpMethod = /./[SYMBOL];\r\n    var fns = exec(\r\n      defined,\r\n      SYMBOL,\r\n      ''[KEY],\r\n      function maybeCallNative(nativeMethod, regexp, str, arg2, forceStringMethod) {\r\n        if (regexp.exec === regexpExec) {\r\n          if (DELEGATES_TO_SYMBOL && !forceStringMethod) {\r\n            // The native String method already delegates to @@method (this\r\n            // polyfilled function), leasing to infinite recursion.\r\n            // We avoid it by directly calling the native @@method method.\r\n            return { done: true, value: nativeRegExpMethod.call(regexp, str, arg2) };\r\n          }\r\n          return { done: true, value: nativeMethod.call(str, regexp, arg2) };\r\n        }\r\n        return { done: false };\r\n      }\r\n    );\r\n    var strfn = fns[0];\r\n    var rxfn = fns[1];\r\n\r\n    redefine(String.prototype, KEY, strfn);\r\n    hide(RegExp.prototype, SYMBOL, length == 2\r\n      // 21.2.5.8 RegExp.prototype[@@replace](string, replaceValue)\r\n      // 21.2.5.11 RegExp.prototype[@@split](string, limit)\r\n      ? function (string, arg) { return rxfn.call(string, this, arg); }\r\n      // 21.2.5.6 RegExp.prototype[@@match](string)\r\n      // 21.2.5.9 RegExp.prototype[@@search](string)\r\n      : function (string) { return rxfn.call(string, this); }\r\n    );\r\n  }\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_fix-re-wks.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_flags.js":
/*!************************************************!*\
  !*** ./node_modules/core-js/modules/_flags.js ***!
  \************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// 21.2.5.3 get RegExp.prototype.flags\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nmodule.exports = function () {\r\n  var that = anObject(this);\r\n  var result = '';\r\n  if (that.global) result += 'g';\r\n  if (that.ignoreCase) result += 'i';\r\n  if (that.multiline) result += 'm';\r\n  if (that.unicode) result += 'u';\r\n  if (that.sticky) result += 'y';\r\n  return result;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_flags.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_flatten-into-array.js":
/*!*************************************************************!*\
  !*** ./node_modules/core-js/modules/_flatten-into-array.js ***!
  \*************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// https://tc39.github.io/proposal-flatMap/#sec-FlattenIntoArray\r\nvar isArray = __webpack_require__(/*! ./_is-array */ \"./node_modules/core-js/modules/_is-array.js\");\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nvar ctx = __webpack_require__(/*! ./_ctx */ \"./node_modules/core-js/modules/_ctx.js\");\r\nvar IS_CONCAT_SPREADABLE = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('isConcatSpreadable');\r\n\r\nfunction flattenIntoArray(target, original, source, sourceLen, start, depth, mapper, thisArg) {\r\n  var targetIndex = start;\r\n  var sourceIndex = 0;\r\n  var mapFn = mapper ? ctx(mapper, thisArg, 3) : false;\r\n  var element, spreadable;\r\n\r\n  while (sourceIndex < sourceLen) {\r\n    if (sourceIndex in source) {\r\n      element = mapFn ? mapFn(source[sourceIndex], sourceIndex, original) : source[sourceIndex];\r\n\r\n      spreadable = false;\r\n      if (isObject(element)) {\r\n        spreadable = element[IS_CONCAT_SPREADABLE];\r\n        spreadable = spreadable !== undefined ? !!spreadable : isArray(element);\r\n      }\r\n\r\n      if (spreadable && depth > 0) {\r\n        targetIndex = flattenIntoArray(target, original, element, toLength(element.length), targetIndex, depth - 1) - 1;\r\n      } else {\r\n        if (targetIndex >= 0x1fffffffffffff) throw TypeError();\r\n        target[targetIndex] = element;\r\n      }\r\n\r\n      targetIndex++;\r\n    }\r\n    sourceIndex++;\r\n  }\r\n  return targetIndex;\r\n}\r\n\r\nmodule.exports = flattenIntoArray;\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_flatten-into-array.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_for-of.js":
/*!*************************************************!*\
  !*** ./node_modules/core-js/modules/_for-of.js ***!
  \*************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var ctx = __webpack_require__(/*! ./_ctx */ \"./node_modules/core-js/modules/_ctx.js\");\r\nvar call = __webpack_require__(/*! ./_iter-call */ \"./node_modules/core-js/modules/_iter-call.js\");\r\nvar isArrayIter = __webpack_require__(/*! ./_is-array-iter */ \"./node_modules/core-js/modules/_is-array-iter.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nvar getIterFn = __webpack_require__(/*! ./core.get-iterator-method */ \"./node_modules/core-js/modules/core.get-iterator-method.js\");\r\nvar BREAK = {};\r\nvar RETURN = {};\r\nvar exports = module.exports = function (iterable, entries, fn, that, ITERATOR) {\r\n  var iterFn = ITERATOR ? function () { return iterable; } : getIterFn(iterable);\r\n  var f = ctx(fn, that, entries ? 2 : 1);\r\n  var index = 0;\r\n  var length, step, iterator, result;\r\n  if (typeof iterFn != 'function') throw TypeError(iterable + ' is not iterable!');\r\n  // fast case for arrays with default iterator\r\n  if (isArrayIter(iterFn)) for (length = toLength(iterable.length); length > index; index++) {\r\n    result = entries ? f(anObject(step = iterable[index])[0], step[1]) : f(iterable[index]);\r\n    if (result === BREAK || result === RETURN) return result;\r\n  } else for (iterator = iterFn.call(iterable); !(step = iterator.next()).done;) {\r\n    result = call(iterator, f, step.value, entries);\r\n    if (result === BREAK || result === RETURN) return result;\r\n  }\r\n};\r\nexports.BREAK = BREAK;\r\nexports.RETURN = RETURN;\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_for-of.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_function-to-string.js":
/*!*************************************************************!*\
  !*** ./node_modules/core-js/modules/_function-to-string.js ***!
  \*************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("module.exports = __webpack_require__(/*! ./_shared */ \"./node_modules/core-js/modules/_shared.js\")('native-function-to-string', Function.toString);\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_function-to-string.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_global.js":
/*!*************************************************!*\
  !*** ./node_modules/core-js/modules/_global.js ***!
  \*************************************************/
/***/ (function(module) {

eval("// https://github.com/zloirock/core-js/issues/86#issuecomment-115759028\r\nvar global = module.exports = typeof window != 'undefined' && window.Math == Math\r\n  ? window : typeof self != 'undefined' && self.Math == Math ? self\r\n  // eslint-disable-next-line no-new-func\r\n  : Function('return this')();\r\nif (typeof __g == 'number') __g = global; // eslint-disable-line no-undef\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_global.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_has.js":
/*!**********************************************!*\
  !*** ./node_modules/core-js/modules/_has.js ***!
  \**********************************************/
/***/ (function(module) {

eval("var hasOwnProperty = {}.hasOwnProperty;\r\nmodule.exports = function (it, key) {\r\n  return hasOwnProperty.call(it, key);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_has.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_hide.js":
/*!***********************************************!*\
  !*** ./node_modules/core-js/modules/_hide.js ***!
  \***********************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var dP = __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\");\r\nvar createDesc = __webpack_require__(/*! ./_property-desc */ \"./node_modules/core-js/modules/_property-desc.js\");\r\nmodule.exports = __webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\") ? function (object, key, value) {\r\n  return dP.f(object, key, createDesc(1, value));\r\n} : function (object, key, value) {\r\n  object[key] = value;\r\n  return object;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_hide.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_html.js":
/*!***********************************************!*\
  !*** ./node_modules/core-js/modules/_html.js ***!
  \***********************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var document = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\").document;\r\nmodule.exports = document && document.documentElement;\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_html.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_ie8-dom-define.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/_ie8-dom-define.js ***!
  \*********************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("module.exports = !__webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\") && !__webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\")(function () {\r\n  return Object.defineProperty(__webpack_require__(/*! ./_dom-create */ \"./node_modules/core-js/modules/_dom-create.js\")('div'), 'a', { get: function () { return 7; } }).a != 7;\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_ie8-dom-define.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_inherit-if-required.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/_inherit-if-required.js ***!
  \**************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar setPrototypeOf = __webpack_require__(/*! ./_set-proto */ \"./node_modules/core-js/modules/_set-proto.js\").set;\r\nmodule.exports = function (that, target, C) {\r\n  var S = target.constructor;\r\n  var P;\r\n  if (S !== C && typeof S == 'function' && (P = S.prototype) !== C.prototype && isObject(P) && setPrototypeOf) {\r\n    setPrototypeOf(that, P);\r\n  } return that;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_inherit-if-required.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_invoke.js":
/*!*************************************************!*\
  !*** ./node_modules/core-js/modules/_invoke.js ***!
  \*************************************************/
/***/ (function(module) {

eval("// fast apply, http://jsperf.lnkit.com/fast-apply/5\r\nmodule.exports = function (fn, args, that) {\r\n  var un = that === undefined;\r\n  switch (args.length) {\r\n    case 0: return un ? fn()\r\n                      : fn.call(that);\r\n    case 1: return un ? fn(args[0])\r\n                      : fn.call(that, args[0]);\r\n    case 2: return un ? fn(args[0], args[1])\r\n                      : fn.call(that, args[0], args[1]);\r\n    case 3: return un ? fn(args[0], args[1], args[2])\r\n                      : fn.call(that, args[0], args[1], args[2]);\r\n    case 4: return un ? fn(args[0], args[1], args[2], args[3])\r\n                      : fn.call(that, args[0], args[1], args[2], args[3]);\r\n  } return fn.apply(that, args);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_invoke.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_iobject.js":
/*!**************************************************!*\
  !*** ./node_modules/core-js/modules/_iobject.js ***!
  \**************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// fallback for non-array-like ES3 and non-enumerable old V8 strings\r\nvar cof = __webpack_require__(/*! ./_cof */ \"./node_modules/core-js/modules/_cof.js\");\r\n// eslint-disable-next-line no-prototype-builtins\r\nmodule.exports = Object('z').propertyIsEnumerable(0) ? Object : function (it) {\r\n  return cof(it) == 'String' ? it.split('') : Object(it);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_iobject.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_is-array-iter.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/_is-array-iter.js ***!
  \********************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// check on default Array iterator\r\nvar Iterators = __webpack_require__(/*! ./_iterators */ \"./node_modules/core-js/modules/_iterators.js\");\r\nvar ITERATOR = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('iterator');\r\nvar ArrayProto = Array.prototype;\r\n\r\nmodule.exports = function (it) {\r\n  return it !== undefined && (Iterators.Array === it || ArrayProto[ITERATOR] === it);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_is-array-iter.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_is-array.js":
/*!***************************************************!*\
  !*** ./node_modules/core-js/modules/_is-array.js ***!
  \***************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// 7.2.2 IsArray(argument)\r\nvar cof = __webpack_require__(/*! ./_cof */ \"./node_modules/core-js/modules/_cof.js\");\r\nmodule.exports = Array.isArray || function isArray(arg) {\r\n  return cof(arg) == 'Array';\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_is-array.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_is-integer.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_is-integer.js ***!
  \*****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.1.2.3 Number.isInteger(number)\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar floor = Math.floor;\r\nmodule.exports = function isInteger(it) {\r\n  return !isObject(it) && isFinite(it) && floor(it) === it;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_is-integer.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_is-object.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/_is-object.js ***!
  \****************************************************/
/***/ (function(module) {

eval("module.exports = function (it) {\r\n  return typeof it === 'object' ? it !== null : typeof it === 'function';\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_is-object.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_is-regexp.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/_is-regexp.js ***!
  \****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// 7.2.8 IsRegExp(argument)\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar cof = __webpack_require__(/*! ./_cof */ \"./node_modules/core-js/modules/_cof.js\");\r\nvar MATCH = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('match');\r\nmodule.exports = function (it) {\r\n  var isRegExp;\r\n  return isObject(it) && ((isRegExp = it[MATCH]) !== undefined ? !!isRegExp : cof(it) == 'RegExp');\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_is-regexp.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_iter-call.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/_iter-call.js ***!
  \****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// call something on iterator step with safe closing on error\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nmodule.exports = function (iterator, fn, value, entries) {\r\n  try {\r\n    return entries ? fn(anObject(value)[0], value[1]) : fn(value);\r\n  // 7.4.6 IteratorClose(iterator, completion)\r\n  } catch (e) {\r\n    var ret = iterator['return'];\r\n    if (ret !== undefined) anObject(ret.call(iterator));\r\n    throw e;\r\n  }\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_iter-call.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_iter-create.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/_iter-create.js ***!
  \******************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar create = __webpack_require__(/*! ./_object-create */ \"./node_modules/core-js/modules/_object-create.js\");\r\nvar descriptor = __webpack_require__(/*! ./_property-desc */ \"./node_modules/core-js/modules/_property-desc.js\");\r\nvar setToStringTag = __webpack_require__(/*! ./_set-to-string-tag */ \"./node_modules/core-js/modules/_set-to-string-tag.js\");\r\nvar IteratorPrototype = {};\r\n\r\n// 25.1.2.1.1 %IteratorPrototype%[@@iterator]()\r\n__webpack_require__(/*! ./_hide */ \"./node_modules/core-js/modules/_hide.js\")(IteratorPrototype, __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('iterator'), function () { return this; });\r\n\r\nmodule.exports = function (Constructor, NAME, next) {\r\n  Constructor.prototype = create(IteratorPrototype, { next: descriptor(1, next) });\r\n  setToStringTag(Constructor, NAME + ' Iterator');\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_iter-create.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_iter-define.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/_iter-define.js ***!
  \******************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar LIBRARY = __webpack_require__(/*! ./_library */ \"./node_modules/core-js/modules/_library.js\");\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar redefine = __webpack_require__(/*! ./_redefine */ \"./node_modules/core-js/modules/_redefine.js\");\r\nvar hide = __webpack_require__(/*! ./_hide */ \"./node_modules/core-js/modules/_hide.js\");\r\nvar Iterators = __webpack_require__(/*! ./_iterators */ \"./node_modules/core-js/modules/_iterators.js\");\r\nvar $iterCreate = __webpack_require__(/*! ./_iter-create */ \"./node_modules/core-js/modules/_iter-create.js\");\r\nvar setToStringTag = __webpack_require__(/*! ./_set-to-string-tag */ \"./node_modules/core-js/modules/_set-to-string-tag.js\");\r\nvar getPrototypeOf = __webpack_require__(/*! ./_object-gpo */ \"./node_modules/core-js/modules/_object-gpo.js\");\r\nvar ITERATOR = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('iterator');\r\nvar BUGGY = !([].keys && 'next' in [].keys()); // Safari has buggy iterators w/o `next`\r\nvar FF_ITERATOR = '@@iterator';\r\nvar KEYS = 'keys';\r\nvar VALUES = 'values';\r\n\r\nvar returnThis = function () { return this; };\r\n\r\nmodule.exports = function (Base, NAME, Constructor, next, DEFAULT, IS_SET, FORCED) {\r\n  $iterCreate(Constructor, NAME, next);\r\n  var getMethod = function (kind) {\r\n    if (!BUGGY && kind in proto) return proto[kind];\r\n    switch (kind) {\r\n      case KEYS: return function keys() { return new Constructor(this, kind); };\r\n      case VALUES: return function values() { return new Constructor(this, kind); };\r\n    } return function entries() { return new Constructor(this, kind); };\r\n  };\r\n  var TAG = NAME + ' Iterator';\r\n  var DEF_VALUES = DEFAULT == VALUES;\r\n  var VALUES_BUG = false;\r\n  var proto = Base.prototype;\r\n  var $native = proto[ITERATOR] || proto[FF_ITERATOR] || DEFAULT && proto[DEFAULT];\r\n  var $default = $native || getMethod(DEFAULT);\r\n  var $entries = DEFAULT ? !DEF_VALUES ? $default : getMethod('entries') : undefined;\r\n  var $anyNative = NAME == 'Array' ? proto.entries || $native : $native;\r\n  var methods, key, IteratorPrototype;\r\n  // Fix native\r\n  if ($anyNative) {\r\n    IteratorPrototype = getPrototypeOf($anyNative.call(new Base()));\r\n    if (IteratorPrototype !== Object.prototype && IteratorPrototype.next) {\r\n      // Set @@toStringTag to native iterators\r\n      setToStringTag(IteratorPrototype, TAG, true);\r\n      // fix for some old engines\r\n      if (!LIBRARY && typeof IteratorPrototype[ITERATOR] != 'function') hide(IteratorPrototype, ITERATOR, returnThis);\r\n    }\r\n  }\r\n  // fix Array#{values, @@iterator}.name in V8 / FF\r\n  if (DEF_VALUES && $native && $native.name !== VALUES) {\r\n    VALUES_BUG = true;\r\n    $default = function values() { return $native.call(this); };\r\n  }\r\n  // Define iterator\r\n  if ((!LIBRARY || FORCED) && (BUGGY || VALUES_BUG || !proto[ITERATOR])) {\r\n    hide(proto, ITERATOR, $default);\r\n  }\r\n  // Plug for library\r\n  Iterators[NAME] = $default;\r\n  Iterators[TAG] = returnThis;\r\n  if (DEFAULT) {\r\n    methods = {\r\n      values: DEF_VALUES ? $default : getMethod(VALUES),\r\n      keys: IS_SET ? $default : getMethod(KEYS),\r\n      entries: $entries\r\n    };\r\n    if (FORCED) for (key in methods) {\r\n      if (!(key in proto)) redefine(proto, key, methods[key]);\r\n    } else $export($export.P + $export.F * (BUGGY || VALUES_BUG), NAME, methods);\r\n  }\r\n  return methods;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_iter-define.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_iter-detect.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/_iter-detect.js ***!
  \******************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var ITERATOR = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('iterator');\r\nvar SAFE_CLOSING = false;\r\n\r\ntry {\r\n  var riter = [7][ITERATOR]();\r\n  riter['return'] = function () { SAFE_CLOSING = true; };\r\n  // eslint-disable-next-line no-throw-literal\r\n  Array.from(riter, function () { throw 2; });\r\n} catch (e) { /* empty */ }\r\n\r\nmodule.exports = function (exec, skipClosing) {\r\n  if (!skipClosing && !SAFE_CLOSING) return false;\r\n  var safe = false;\r\n  try {\r\n    var arr = [7];\r\n    var iter = arr[ITERATOR]();\r\n    iter.next = function () { return { done: safe = true }; };\r\n    arr[ITERATOR] = function () { return iter; };\r\n    exec(arr);\r\n  } catch (e) { /* empty */ }\r\n  return safe;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_iter-detect.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_iter-step.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/_iter-step.js ***!
  \****************************************************/
/***/ (function(module) {

eval("module.exports = function (done, value) {\r\n  return { value: value, done: !!done };\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_iter-step.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_iterators.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/_iterators.js ***!
  \****************************************************/
/***/ (function(module) {

eval("module.exports = {};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_iterators.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_library.js":
/*!**************************************************!*\
  !*** ./node_modules/core-js/modules/_library.js ***!
  \**************************************************/
/***/ (function(module) {

eval("module.exports = false;\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_library.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_math-expm1.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_math-expm1.js ***!
  \*****************************************************/
/***/ (function(module) {

eval("// 20.2.2.14 Math.expm1(x)\r\nvar $expm1 = Math.expm1;\r\nmodule.exports = (!$expm1\r\n  // Old FF bug\r\n  || $expm1(10) > 22025.465794806719 || $expm1(10) < 22025.4657948067165168\r\n  // Tor Browser bug\r\n  || $expm1(-2e-17) != -2e-17\r\n) ? function expm1(x) {\r\n  return (x = +x) == 0 ? x : x > -1e-6 && x < 1e-6 ? x + x * x / 2 : Math.exp(x) - 1;\r\n} : $expm1;\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_math-expm1.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_math-fround.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/_math-fround.js ***!
  \******************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.2.2.16 Math.fround(x)\r\nvar sign = __webpack_require__(/*! ./_math-sign */ \"./node_modules/core-js/modules/_math-sign.js\");\r\nvar pow = Math.pow;\r\nvar EPSILON = pow(2, -52);\r\nvar EPSILON32 = pow(2, -23);\r\nvar MAX32 = pow(2, 127) * (2 - EPSILON32);\r\nvar MIN32 = pow(2, -126);\r\n\r\nvar roundTiesToEven = function (n) {\r\n  return n + 1 / EPSILON - 1 / EPSILON;\r\n};\r\n\r\nmodule.exports = Math.fround || function fround(x) {\r\n  var $abs = Math.abs(x);\r\n  var $sign = sign(x);\r\n  var a, result;\r\n  if ($abs < MIN32) return $sign * roundTiesToEven($abs / MIN32 / EPSILON32) * MIN32 * EPSILON32;\r\n  a = (1 + EPSILON32 / EPSILON) * $abs;\r\n  result = a - (a - $abs);\r\n  // eslint-disable-next-line no-self-compare\r\n  if (result > MAX32 || result != result) return $sign * Infinity;\r\n  return $sign * result;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_math-fround.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_math-log1p.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_math-log1p.js ***!
  \*****************************************************/
/***/ (function(module) {

eval("// 20.2.2.20 Math.log1p(x)\r\nmodule.exports = Math.log1p || function log1p(x) {\r\n  return (x = +x) > -1e-8 && x < 1e-8 ? x - x * x / 2 : Math.log(1 + x);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_math-log1p.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_math-scale.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_math-scale.js ***!
  \*****************************************************/
/***/ (function(module) {

eval("// https://rwaldron.github.io/proposal-math-extensions/\r\nmodule.exports = Math.scale || function scale(x, inLow, inHigh, outLow, outHigh) {\r\n  if (\r\n    arguments.length === 0\r\n      // eslint-disable-next-line no-self-compare\r\n      || x != x\r\n      // eslint-disable-next-line no-self-compare\r\n      || inLow != inLow\r\n      // eslint-disable-next-line no-self-compare\r\n      || inHigh != inHigh\r\n      // eslint-disable-next-line no-self-compare\r\n      || outLow != outLow\r\n      // eslint-disable-next-line no-self-compare\r\n      || outHigh != outHigh\r\n  ) return NaN;\r\n  if (x === Infinity || x === -Infinity) return x;\r\n  return (x - inLow) * (outHigh - outLow) / (inHigh - inLow) + outLow;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_math-scale.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_math-sign.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/_math-sign.js ***!
  \****************************************************/
/***/ (function(module) {

eval("// 20.2.2.28 Math.sign(x)\r\nmodule.exports = Math.sign || function sign(x) {\r\n  // eslint-disable-next-line no-self-compare\r\n  return (x = +x) == 0 || x != x ? x : x < 0 ? -1 : 1;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_math-sign.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_meta.js":
/*!***********************************************!*\
  !*** ./node_modules/core-js/modules/_meta.js ***!
  \***********************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var META = __webpack_require__(/*! ./_uid */ \"./node_modules/core-js/modules/_uid.js\")('meta');\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar has = __webpack_require__(/*! ./_has */ \"./node_modules/core-js/modules/_has.js\");\r\nvar setDesc = __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\").f;\r\nvar id = 0;\r\nvar isExtensible = Object.isExtensible || function () {\r\n  return true;\r\n};\r\nvar FREEZE = !__webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\")(function () {\r\n  return isExtensible(Object.preventExtensions({}));\r\n});\r\nvar setMeta = function (it) {\r\n  setDesc(it, META, { value: {\r\n    i: 'O' + ++id, // object ID\r\n    w: {}          // weak collections IDs\r\n  } });\r\n};\r\nvar fastKey = function (it, create) {\r\n  // return primitive with prefix\r\n  if (!isObject(it)) return typeof it == 'symbol' ? it : (typeof it == 'string' ? 'S' : 'P') + it;\r\n  if (!has(it, META)) {\r\n    // can't set metadata to uncaught frozen object\r\n    if (!isExtensible(it)) return 'F';\r\n    // not necessary to add metadata\r\n    if (!create) return 'E';\r\n    // add missing metadata\r\n    setMeta(it);\r\n  // return object ID\r\n  } return it[META].i;\r\n};\r\nvar getWeak = function (it, create) {\r\n  if (!has(it, META)) {\r\n    // can't set metadata to uncaught frozen object\r\n    if (!isExtensible(it)) return true;\r\n    // not necessary to add metadata\r\n    if (!create) return false;\r\n    // add missing metadata\r\n    setMeta(it);\r\n  // return hash weak collections IDs\r\n  } return it[META].w;\r\n};\r\n// add metadata on freeze-family methods calling\r\nvar onFreeze = function (it) {\r\n  if (FREEZE && meta.NEED && isExtensible(it) && !has(it, META)) setMeta(it);\r\n  return it;\r\n};\r\nvar meta = module.exports = {\r\n  KEY: META,\r\n  NEED: false,\r\n  fastKey: fastKey,\r\n  getWeak: getWeak,\r\n  onFreeze: onFreeze\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_meta.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_metadata.js":
/*!***************************************************!*\
  !*** ./node_modules/core-js/modules/_metadata.js ***!
  \***************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var Map = __webpack_require__(/*! ./es6.map */ \"./node_modules/core-js/modules/es6.map.js\");\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar shared = __webpack_require__(/*! ./_shared */ \"./node_modules/core-js/modules/_shared.js\")('metadata');\r\nvar store = shared.store || (shared.store = new (__webpack_require__(/*! ./es6.weak-map */ \"./node_modules/core-js/modules/es6.weak-map.js\"))());\r\n\r\nvar getOrCreateMetadataMap = function (target, targetKey, create) {\r\n  var targetMetadata = store.get(target);\r\n  if (!targetMetadata) {\r\n    if (!create) return undefined;\r\n    store.set(target, targetMetadata = new Map());\r\n  }\r\n  var keyMetadata = targetMetadata.get(targetKey);\r\n  if (!keyMetadata) {\r\n    if (!create) return undefined;\r\n    targetMetadata.set(targetKey, keyMetadata = new Map());\r\n  } return keyMetadata;\r\n};\r\nvar ordinaryHasOwnMetadata = function (MetadataKey, O, P) {\r\n  var metadataMap = getOrCreateMetadataMap(O, P, false);\r\n  return metadataMap === undefined ? false : metadataMap.has(MetadataKey);\r\n};\r\nvar ordinaryGetOwnMetadata = function (MetadataKey, O, P) {\r\n  var metadataMap = getOrCreateMetadataMap(O, P, false);\r\n  return metadataMap === undefined ? undefined : metadataMap.get(MetadataKey);\r\n};\r\nvar ordinaryDefineOwnMetadata = function (MetadataKey, MetadataValue, O, P) {\r\n  getOrCreateMetadataMap(O, P, true).set(MetadataKey, MetadataValue);\r\n};\r\nvar ordinaryOwnMetadataKeys = function (target, targetKey) {\r\n  var metadataMap = getOrCreateMetadataMap(target, targetKey, false);\r\n  var keys = [];\r\n  if (metadataMap) metadataMap.forEach(function (_, key) { keys.push(key); });\r\n  return keys;\r\n};\r\nvar toMetaKey = function (it) {\r\n  return it === undefined || typeof it == 'symbol' ? it : String(it);\r\n};\r\nvar exp = function (O) {\r\n  $export($export.S, 'Reflect', O);\r\n};\r\n\r\nmodule.exports = {\r\n  store: store,\r\n  map: getOrCreateMetadataMap,\r\n  has: ordinaryHasOwnMetadata,\r\n  get: ordinaryGetOwnMetadata,\r\n  set: ordinaryDefineOwnMetadata,\r\n  keys: ordinaryOwnMetadataKeys,\r\n  key: toMetaKey,\r\n  exp: exp\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_metadata.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_microtask.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/_microtask.js ***!
  \****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar macrotask = __webpack_require__(/*! ./_task */ \"./node_modules/core-js/modules/_task.js\").set;\r\nvar Observer = global.MutationObserver || global.WebKitMutationObserver;\r\nvar process = global.process;\r\nvar Promise = global.Promise;\r\nvar isNode = __webpack_require__(/*! ./_cof */ \"./node_modules/core-js/modules/_cof.js\")(process) == 'process';\r\n\r\nmodule.exports = function () {\r\n  var head, last, notify;\r\n\r\n  var flush = function () {\r\n    var parent, fn;\r\n    if (isNode && (parent = process.domain)) parent.exit();\r\n    while (head) {\r\n      fn = head.fn;\r\n      head = head.next;\r\n      try {\r\n        fn();\r\n      } catch (e) {\r\n        if (head) notify();\r\n        else last = undefined;\r\n        throw e;\r\n      }\r\n    } last = undefined;\r\n    if (parent) parent.enter();\r\n  };\r\n\r\n  // Node.js\r\n  if (isNode) {\r\n    notify = function () {\r\n      process.nextTick(flush);\r\n    };\r\n  // browsers with MutationObserver, except iOS Safari - https://github.com/zloirock/core-js/issues/339\r\n  } else if (Observer && !(global.navigator && global.navigator.standalone)) {\r\n    var toggle = true;\r\n    var node = document.createTextNode('');\r\n    new Observer(flush).observe(node, { characterData: true }); // eslint-disable-line no-new\r\n    notify = function () {\r\n      node.data = toggle = !toggle;\r\n    };\r\n  // environments with maybe non-completely correct, but existent Promise\r\n  } else if (Promise && Promise.resolve) {\r\n    // Promise.resolve without an argument throws an error in LG WebOS 2\r\n    var promise = Promise.resolve(undefined);\r\n    notify = function () {\r\n      promise.then(flush);\r\n    };\r\n  // for other environments - macrotask based on:\r\n  // - setImmediate\r\n  // - MessageChannel\r\n  // - window.postMessag\r\n  // - onreadystatechange\r\n  // - setTimeout\r\n  } else {\r\n    notify = function () {\r\n      // strange IE + webpack dev server bug - use .call(global)\r\n      macrotask.call(global, flush);\r\n    };\r\n  }\r\n\r\n  return function (fn) {\r\n    var task = { fn: fn, next: undefined };\r\n    if (last) last.next = task;\r\n    if (!head) {\r\n      head = task;\r\n      notify();\r\n    } last = task;\r\n  };\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_microtask.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_new-promise-capability.js":
/*!*****************************************************************!*\
  !*** ./node_modules/core-js/modules/_new-promise-capability.js ***!
  \*****************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// 25.4.1.5 NewPromiseCapability(C)\r\nvar aFunction = __webpack_require__(/*! ./_a-function */ \"./node_modules/core-js/modules/_a-function.js\");\r\n\r\nfunction PromiseCapability(C) {\r\n  var resolve, reject;\r\n  this.promise = new C(function ($$resolve, $$reject) {\r\n    if (resolve !== undefined || reject !== undefined) throw TypeError('Bad Promise constructor');\r\n    resolve = $$resolve;\r\n    reject = $$reject;\r\n  });\r\n  this.resolve = aFunction(resolve);\r\n  this.reject = aFunction(reject);\r\n}\r\n\r\nmodule.exports.f = function (C) {\r\n  return new PromiseCapability(C);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_new-promise-capability.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_object-assign.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/_object-assign.js ***!
  \********************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// 19.1.2.1 Object.assign(target, source, ...)\r\nvar DESCRIPTORS = __webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\");\r\nvar getKeys = __webpack_require__(/*! ./_object-keys */ \"./node_modules/core-js/modules/_object-keys.js\");\r\nvar gOPS = __webpack_require__(/*! ./_object-gops */ \"./node_modules/core-js/modules/_object-gops.js\");\r\nvar pIE = __webpack_require__(/*! ./_object-pie */ \"./node_modules/core-js/modules/_object-pie.js\");\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar IObject = __webpack_require__(/*! ./_iobject */ \"./node_modules/core-js/modules/_iobject.js\");\r\nvar $assign = Object.assign;\r\n\r\n// should work with symbols and should have deterministic property order (V8 bug)\r\nmodule.exports = !$assign || __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\")(function () {\r\n  var A = {};\r\n  var B = {};\r\n  // eslint-disable-next-line no-undef\r\n  var S = Symbol();\r\n  var K = 'abcdefghijklmnopqrst';\r\n  A[S] = 7;\r\n  K.split('').forEach(function (k) { B[k] = k; });\r\n  return $assign({}, A)[S] != 7 || Object.keys($assign({}, B)).join('') != K;\r\n}) ? function assign(target, source) { // eslint-disable-line no-unused-vars\r\n  var T = toObject(target);\r\n  var aLen = arguments.length;\r\n  var index = 1;\r\n  var getSymbols = gOPS.f;\r\n  var isEnum = pIE.f;\r\n  while (aLen > index) {\r\n    var S = IObject(arguments[index++]);\r\n    var keys = getSymbols ? getKeys(S).concat(getSymbols(S)) : getKeys(S);\r\n    var length = keys.length;\r\n    var j = 0;\r\n    var key;\r\n    while (length > j) {\r\n      key = keys[j++];\r\n      if (!DESCRIPTORS || isEnum.call(S, key)) T[key] = S[key];\r\n    }\r\n  } return T;\r\n} : $assign;\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_object-assign.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_object-create.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/_object-create.js ***!
  \********************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// 19.1.2.2 / 15.2.3.5 Object.create(O [, Properties])\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar dPs = __webpack_require__(/*! ./_object-dps */ \"./node_modules/core-js/modules/_object-dps.js\");\r\nvar enumBugKeys = __webpack_require__(/*! ./_enum-bug-keys */ \"./node_modules/core-js/modules/_enum-bug-keys.js\");\r\nvar IE_PROTO = __webpack_require__(/*! ./_shared-key */ \"./node_modules/core-js/modules/_shared-key.js\")('IE_PROTO');\r\nvar Empty = function () { /* empty */ };\r\nvar PROTOTYPE = 'prototype';\r\n\r\n// Create object with fake `null` prototype: use iframe Object with cleared prototype\r\nvar createDict = function () {\r\n  // Thrash, waste and sodomy: IE GC bug\r\n  var iframe = __webpack_require__(/*! ./_dom-create */ \"./node_modules/core-js/modules/_dom-create.js\")('iframe');\r\n  var i = enumBugKeys.length;\r\n  var lt = '<';\r\n  var gt = '>';\r\n  var iframeDocument;\r\n  iframe.style.display = 'none';\r\n  __webpack_require__(/*! ./_html */ \"./node_modules/core-js/modules/_html.js\").appendChild(iframe);\r\n  iframe.src = 'javascript:'; // eslint-disable-line no-script-url\r\n  // createDict = iframe.contentWindow.Object;\r\n  // html.removeChild(iframe);\r\n  iframeDocument = iframe.contentWindow.document;\r\n  iframeDocument.open();\r\n  iframeDocument.write(lt + 'script' + gt + 'document.F=Object' + lt + '/script' + gt);\r\n  iframeDocument.close();\r\n  createDict = iframeDocument.F;\r\n  while (i--) delete createDict[PROTOTYPE][enumBugKeys[i]];\r\n  return createDict();\r\n};\r\n\r\nmodule.exports = Object.create || function create(O, Properties) {\r\n  var result;\r\n  if (O !== null) {\r\n    Empty[PROTOTYPE] = anObject(O);\r\n    result = new Empty();\r\n    Empty[PROTOTYPE] = null;\r\n    // add \"__proto__\" for Object.getPrototypeOf polyfill\r\n    result[IE_PROTO] = O;\r\n  } else result = createDict();\r\n  return Properties === undefined ? result : dPs(result, Properties);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_object-create.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_object-dp.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/_object-dp.js ***!
  \****************************************************/
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {

eval("var anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar IE8_DOM_DEFINE = __webpack_require__(/*! ./_ie8-dom-define */ \"./node_modules/core-js/modules/_ie8-dom-define.js\");\r\nvar toPrimitive = __webpack_require__(/*! ./_to-primitive */ \"./node_modules/core-js/modules/_to-primitive.js\");\r\nvar dP = Object.defineProperty;\r\n\r\nexports.f = __webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\") ? Object.defineProperty : function defineProperty(O, P, Attributes) {\r\n  anObject(O);\r\n  P = toPrimitive(P, true);\r\n  anObject(Attributes);\r\n  if (IE8_DOM_DEFINE) try {\r\n    return dP(O, P, Attributes);\r\n  } catch (e) { /* empty */ }\r\n  if ('get' in Attributes || 'set' in Attributes) throw TypeError('Accessors not supported!');\r\n  if ('value' in Attributes) O[P] = Attributes.value;\r\n  return O;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_object-dp.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_object-dps.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_object-dps.js ***!
  \*****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var dP = __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar getKeys = __webpack_require__(/*! ./_object-keys */ \"./node_modules/core-js/modules/_object-keys.js\");\r\n\r\nmodule.exports = __webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\") ? Object.defineProperties : function defineProperties(O, Properties) {\r\n  anObject(O);\r\n  var keys = getKeys(Properties);\r\n  var length = keys.length;\r\n  var i = 0;\r\n  var P;\r\n  while (length > i) dP.f(O, P = keys[i++], Properties[P]);\r\n  return O;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_object-dps.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_object-forced-pam.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/_object-forced-pam.js ***!
  \************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// Forced replacement prototype accessors methods\r\nmodule.exports = __webpack_require__(/*! ./_library */ \"./node_modules/core-js/modules/_library.js\") || !__webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\")(function () {\r\n  var K = Math.random();\r\n  // In FF throws only define methods\r\n  // eslint-disable-next-line no-undef, no-useless-call\r\n  __defineSetter__.call(null, K, function () { /* empty */ });\r\n  delete __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\")[K];\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_object-forced-pam.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_object-gopd.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/_object-gopd.js ***!
  \******************************************************/
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {

eval("var pIE = __webpack_require__(/*! ./_object-pie */ \"./node_modules/core-js/modules/_object-pie.js\");\r\nvar createDesc = __webpack_require__(/*! ./_property-desc */ \"./node_modules/core-js/modules/_property-desc.js\");\r\nvar toIObject = __webpack_require__(/*! ./_to-iobject */ \"./node_modules/core-js/modules/_to-iobject.js\");\r\nvar toPrimitive = __webpack_require__(/*! ./_to-primitive */ \"./node_modules/core-js/modules/_to-primitive.js\");\r\nvar has = __webpack_require__(/*! ./_has */ \"./node_modules/core-js/modules/_has.js\");\r\nvar IE8_DOM_DEFINE = __webpack_require__(/*! ./_ie8-dom-define */ \"./node_modules/core-js/modules/_ie8-dom-define.js\");\r\nvar gOPD = Object.getOwnPropertyDescriptor;\r\n\r\nexports.f = __webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\") ? gOPD : function getOwnPropertyDescriptor(O, P) {\r\n  O = toIObject(O);\r\n  P = toPrimitive(P, true);\r\n  if (IE8_DOM_DEFINE) try {\r\n    return gOPD(O, P);\r\n  } catch (e) { /* empty */ }\r\n  if (has(O, P)) return createDesc(!pIE.f.call(O, P), O[P]);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_object-gopd.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_object-gopn-ext.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/_object-gopn-ext.js ***!
  \**********************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// fallback for IE11 buggy Object.getOwnPropertyNames with iframe and window\r\nvar toIObject = __webpack_require__(/*! ./_to-iobject */ \"./node_modules/core-js/modules/_to-iobject.js\");\r\nvar gOPN = __webpack_require__(/*! ./_object-gopn */ \"./node_modules/core-js/modules/_object-gopn.js\").f;\r\nvar toString = {}.toString;\r\n\r\nvar windowNames = typeof window == 'object' && window && Object.getOwnPropertyNames\r\n  ? Object.getOwnPropertyNames(window) : [];\r\n\r\nvar getWindowNames = function (it) {\r\n  try {\r\n    return gOPN(it);\r\n  } catch (e) {\r\n    return windowNames.slice();\r\n  }\r\n};\r\n\r\nmodule.exports.f = function getOwnPropertyNames(it) {\r\n  return windowNames && toString.call(it) == '[object Window]' ? getWindowNames(it) : gOPN(toIObject(it));\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_object-gopn-ext.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_object-gopn.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/_object-gopn.js ***!
  \******************************************************/
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {

eval("// 19.1.2.7 / 15.2.3.4 Object.getOwnPropertyNames(O)\r\nvar $keys = __webpack_require__(/*! ./_object-keys-internal */ \"./node_modules/core-js/modules/_object-keys-internal.js\");\r\nvar hiddenKeys = __webpack_require__(/*! ./_enum-bug-keys */ \"./node_modules/core-js/modules/_enum-bug-keys.js\").concat('length', 'prototype');\r\n\r\nexports.f = Object.getOwnPropertyNames || function getOwnPropertyNames(O) {\r\n  return $keys(O, hiddenKeys);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_object-gopn.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_object-gops.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/_object-gops.js ***!
  \******************************************************/
/***/ (function(__unused_webpack_module, exports) {

eval("exports.f = Object.getOwnPropertySymbols;\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_object-gops.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_object-gpo.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_object-gpo.js ***!
  \*****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// 19.1.2.9 / 15.2.3.2 Object.getPrototypeOf(O)\r\nvar has = __webpack_require__(/*! ./_has */ \"./node_modules/core-js/modules/_has.js\");\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar IE_PROTO = __webpack_require__(/*! ./_shared-key */ \"./node_modules/core-js/modules/_shared-key.js\")('IE_PROTO');\r\nvar ObjectProto = Object.prototype;\r\n\r\nmodule.exports = Object.getPrototypeOf || function (O) {\r\n  O = toObject(O);\r\n  if (has(O, IE_PROTO)) return O[IE_PROTO];\r\n  if (typeof O.constructor == 'function' && O instanceof O.constructor) {\r\n    return O.constructor.prototype;\r\n  } return O instanceof Object ? ObjectProto : null;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_object-gpo.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_object-keys-internal.js":
/*!***************************************************************!*\
  !*** ./node_modules/core-js/modules/_object-keys-internal.js ***!
  \***************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var has = __webpack_require__(/*! ./_has */ \"./node_modules/core-js/modules/_has.js\");\r\nvar toIObject = __webpack_require__(/*! ./_to-iobject */ \"./node_modules/core-js/modules/_to-iobject.js\");\r\nvar arrayIndexOf = __webpack_require__(/*! ./_array-includes */ \"./node_modules/core-js/modules/_array-includes.js\")(false);\r\nvar IE_PROTO = __webpack_require__(/*! ./_shared-key */ \"./node_modules/core-js/modules/_shared-key.js\")('IE_PROTO');\r\n\r\nmodule.exports = function (object, names) {\r\n  var O = toIObject(object);\r\n  var i = 0;\r\n  var result = [];\r\n  var key;\r\n  for (key in O) if (key != IE_PROTO) has(O, key) && result.push(key);\r\n  // Don't enum bug & hidden keys\r\n  while (names.length > i) if (has(O, key = names[i++])) {\r\n    ~arrayIndexOf(result, key) || result.push(key);\r\n  }\r\n  return result;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_object-keys-internal.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_object-keys.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/_object-keys.js ***!
  \******************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// 19.1.2.14 / 15.2.3.14 Object.keys(O)\r\nvar $keys = __webpack_require__(/*! ./_object-keys-internal */ \"./node_modules/core-js/modules/_object-keys-internal.js\");\r\nvar enumBugKeys = __webpack_require__(/*! ./_enum-bug-keys */ \"./node_modules/core-js/modules/_enum-bug-keys.js\");\r\n\r\nmodule.exports = Object.keys || function keys(O) {\r\n  return $keys(O, enumBugKeys);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_object-keys.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_object-pie.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_object-pie.js ***!
  \*****************************************************/
/***/ (function(__unused_webpack_module, exports) {

eval("exports.f = {}.propertyIsEnumerable;\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_object-pie.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_object-sap.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_object-sap.js ***!
  \*****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// most Object methods by ES6 should accept primitives\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar core = __webpack_require__(/*! ./_core */ \"./node_modules/core-js/modules/_core.js\");\r\nvar fails = __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\");\r\nmodule.exports = function (KEY, exec) {\r\n  var fn = (core.Object || {})[KEY] || Object[KEY];\r\n  var exp = {};\r\n  exp[KEY] = exec(fn);\r\n  $export($export.S + $export.F * fails(function () { fn(1); }), 'Object', exp);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_object-sap.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_object-to-array.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/_object-to-array.js ***!
  \**********************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var DESCRIPTORS = __webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\");\r\nvar getKeys = __webpack_require__(/*! ./_object-keys */ \"./node_modules/core-js/modules/_object-keys.js\");\r\nvar toIObject = __webpack_require__(/*! ./_to-iobject */ \"./node_modules/core-js/modules/_to-iobject.js\");\r\nvar isEnum = __webpack_require__(/*! ./_object-pie */ \"./node_modules/core-js/modules/_object-pie.js\").f;\r\nmodule.exports = function (isEntries) {\r\n  return function (it) {\r\n    var O = toIObject(it);\r\n    var keys = getKeys(O);\r\n    var length = keys.length;\r\n    var i = 0;\r\n    var result = [];\r\n    var key;\r\n    while (length > i) {\r\n      key = keys[i++];\r\n      if (!DESCRIPTORS || isEnum.call(O, key)) {\r\n        result.push(isEntries ? [key, O[key]] : O[key]);\r\n      }\r\n    }\r\n    return result;\r\n  };\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_object-to-array.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_own-keys.js":
/*!***************************************************!*\
  !*** ./node_modules/core-js/modules/_own-keys.js ***!
  \***************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// all object keys, includes non-enumerable and symbols\r\nvar gOPN = __webpack_require__(/*! ./_object-gopn */ \"./node_modules/core-js/modules/_object-gopn.js\");\r\nvar gOPS = __webpack_require__(/*! ./_object-gops */ \"./node_modules/core-js/modules/_object-gops.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar Reflect = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\").Reflect;\r\nmodule.exports = Reflect && Reflect.ownKeys || function ownKeys(it) {\r\n  var keys = gOPN.f(anObject(it));\r\n  var getSymbols = gOPS.f;\r\n  return getSymbols ? keys.concat(getSymbols(it)) : keys;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_own-keys.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_parse-float.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/_parse-float.js ***!
  \******************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var $parseFloat = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\").parseFloat;\r\nvar $trim = __webpack_require__(/*! ./_string-trim */ \"./node_modules/core-js/modules/_string-trim.js\").trim;\r\n\r\nmodule.exports = 1 / $parseFloat(__webpack_require__(/*! ./_string-ws */ \"./node_modules/core-js/modules/_string-ws.js\") + '-0') !== -Infinity ? function parseFloat(str) {\r\n  var string = $trim(String(str), 3);\r\n  var result = $parseFloat(string);\r\n  return result === 0 && string.charAt(0) == '-' ? -0 : result;\r\n} : $parseFloat;\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_parse-float.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_parse-int.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/_parse-int.js ***!
  \****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var $parseInt = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\").parseInt;\r\nvar $trim = __webpack_require__(/*! ./_string-trim */ \"./node_modules/core-js/modules/_string-trim.js\").trim;\r\nvar ws = __webpack_require__(/*! ./_string-ws */ \"./node_modules/core-js/modules/_string-ws.js\");\r\nvar hex = /^[-+]?0[xX]/;\r\n\r\nmodule.exports = $parseInt(ws + '08') !== 8 || $parseInt(ws + '0x16') !== 22 ? function parseInt(str, radix) {\r\n  var string = $trim(String(str), 3);\r\n  return $parseInt(string, (radix >>> 0) || (hex.test(string) ? 16 : 10));\r\n} : $parseInt;\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_parse-int.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_perform.js":
/*!**************************************************!*\
  !*** ./node_modules/core-js/modules/_perform.js ***!
  \**************************************************/
/***/ (function(module) {

eval("module.exports = function (exec) {\r\n  try {\r\n    return { e: false, v: exec() };\r\n  } catch (e) {\r\n    return { e: true, v: e };\r\n  }\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_perform.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_promise-resolve.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/_promise-resolve.js ***!
  \**********************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar newPromiseCapability = __webpack_require__(/*! ./_new-promise-capability */ \"./node_modules/core-js/modules/_new-promise-capability.js\");\r\n\r\nmodule.exports = function (C, x) {\r\n  anObject(C);\r\n  if (isObject(x) && x.constructor === C) return x;\r\n  var promiseCapability = newPromiseCapability.f(C);\r\n  var resolve = promiseCapability.resolve;\r\n  resolve(x);\r\n  return promiseCapability.promise;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_promise-resolve.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_property-desc.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/_property-desc.js ***!
  \********************************************************/
/***/ (function(module) {

eval("module.exports = function (bitmap, value) {\r\n  return {\r\n    enumerable: !(bitmap & 1),\r\n    configurable: !(bitmap & 2),\r\n    writable: !(bitmap & 4),\r\n    value: value\r\n  };\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_property-desc.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_redefine-all.js":
/*!*******************************************************!*\
  !*** ./node_modules/core-js/modules/_redefine-all.js ***!
  \*******************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var redefine = __webpack_require__(/*! ./_redefine */ \"./node_modules/core-js/modules/_redefine.js\");\r\nmodule.exports = function (target, src, safe) {\r\n  for (var key in src) redefine(target, key, src[key], safe);\r\n  return target;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_redefine-all.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_redefine.js":
/*!***************************************************!*\
  !*** ./node_modules/core-js/modules/_redefine.js ***!
  \***************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar hide = __webpack_require__(/*! ./_hide */ \"./node_modules/core-js/modules/_hide.js\");\r\nvar has = __webpack_require__(/*! ./_has */ \"./node_modules/core-js/modules/_has.js\");\r\nvar SRC = __webpack_require__(/*! ./_uid */ \"./node_modules/core-js/modules/_uid.js\")('src');\r\nvar $toString = __webpack_require__(/*! ./_function-to-string */ \"./node_modules/core-js/modules/_function-to-string.js\");\r\nvar TO_STRING = 'toString';\r\nvar TPL = ('' + $toString).split(TO_STRING);\r\n\r\n__webpack_require__(/*! ./_core */ \"./node_modules/core-js/modules/_core.js\").inspectSource = function (it) {\r\n  return $toString.call(it);\r\n};\r\n\r\n(module.exports = function (O, key, val, safe) {\r\n  var isFunction = typeof val == 'function';\r\n  if (isFunction) has(val, 'name') || hide(val, 'name', key);\r\n  if (O[key] === val) return;\r\n  if (isFunction) has(val, SRC) || hide(val, SRC, O[key] ? '' + O[key] : TPL.join(String(key)));\r\n  if (O === global) {\r\n    O[key] = val;\r\n  } else if (!safe) {\r\n    delete O[key];\r\n    hide(O, key, val);\r\n  } else if (O[key]) {\r\n    O[key] = val;\r\n  } else {\r\n    hide(O, key, val);\r\n  }\r\n// add fake Function#toString for correct work wrapped methods / constructors with methods like LoDash isNative\r\n})(Function.prototype, TO_STRING, function toString() {\r\n  return typeof this == 'function' && this[SRC] || $toString.call(this);\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_redefine.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_regexp-exec-abstract.js":
/*!***************************************************************!*\
  !*** ./node_modules/core-js/modules/_regexp-exec-abstract.js ***!
  \***************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n\r\nvar classof = __webpack_require__(/*! ./_classof */ \"./node_modules/core-js/modules/_classof.js\");\r\nvar builtinExec = RegExp.prototype.exec;\r\n\r\n // `RegExpExec` abstract operation\r\n// https://tc39.github.io/ecma262/#sec-regexpexec\r\nmodule.exports = function (R, S) {\r\n  var exec = R.exec;\r\n  if (typeof exec === 'function') {\r\n    var result = exec.call(R, S);\r\n    if (typeof result !== 'object') {\r\n      throw new TypeError('RegExp exec method returned something other than an Object or null');\r\n    }\r\n    return result;\r\n  }\r\n  if (classof(R) !== 'RegExp') {\r\n    throw new TypeError('RegExp#exec called on incompatible receiver');\r\n  }\r\n  return builtinExec.call(R, S);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_regexp-exec-abstract.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_regexp-exec.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/_regexp-exec.js ***!
  \******************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n\r\nvar regexpFlags = __webpack_require__(/*! ./_flags */ \"./node_modules/core-js/modules/_flags.js\");\r\n\r\nvar nativeExec = RegExp.prototype.exec;\r\n// This always refers to the native implementation, because the\r\n// String#replace polyfill uses ./fix-regexp-well-known-symbol-logic.js,\r\n// which loads this file before patching the method.\r\nvar nativeReplace = String.prototype.replace;\r\n\r\nvar patchedExec = nativeExec;\r\n\r\nvar LAST_INDEX = 'lastIndex';\r\n\r\nvar UPDATES_LAST_INDEX_WRONG = (function () {\r\n  var re1 = /a/,\r\n      re2 = /b*/g;\r\n  nativeExec.call(re1, 'a');\r\n  nativeExec.call(re2, 'a');\r\n  return re1[LAST_INDEX] !== 0 || re2[LAST_INDEX] !== 0;\r\n})();\r\n\r\n// nonparticipating capturing group, copied from es5-shim's String#split patch.\r\nvar NPCG_INCLUDED = /()??/.exec('')[1] !== undefined;\r\n\r\nvar PATCH = UPDATES_LAST_INDEX_WRONG || NPCG_INCLUDED;\r\n\r\nif (PATCH) {\r\n  patchedExec = function exec(str) {\r\n    var re = this;\r\n    var lastIndex, reCopy, match, i;\r\n\r\n    if (NPCG_INCLUDED) {\r\n      reCopy = new RegExp('^' + re.source + '$(?!\\\\s)', regexpFlags.call(re));\r\n    }\r\n    if (UPDATES_LAST_INDEX_WRONG) lastIndex = re[LAST_INDEX];\r\n\r\n    match = nativeExec.call(re, str);\r\n\r\n    if (UPDATES_LAST_INDEX_WRONG && match) {\r\n      re[LAST_INDEX] = re.global ? match.index + match[0].length : lastIndex;\r\n    }\r\n    if (NPCG_INCLUDED && match && match.length > 1) {\r\n      // Fix browsers whose `exec` methods don't consistently return `undefined`\r\n      // for NPCG, like IE8. NOTE: This doesn' work for /(.?)?/\r\n      // eslint-disable-next-line no-loop-func\r\n      nativeReplace.call(match[0], reCopy, function () {\r\n        for (i = 1; i < arguments.length - 2; i++) {\r\n          if (arguments[i] === undefined) match[i] = undefined;\r\n        }\r\n      });\r\n    }\r\n\r\n    return match;\r\n  };\r\n}\r\n\r\nmodule.exports = patchedExec;\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_regexp-exec.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_replacer.js":
/*!***************************************************!*\
  !*** ./node_modules/core-js/modules/_replacer.js ***!
  \***************************************************/
/***/ (function(module) {

eval("module.exports = function (regExp, replace) {\r\n  var replacer = replace === Object(replace) ? function (part) {\r\n    return replace[part];\r\n  } : replace;\r\n  return function (it) {\r\n    return String(it).replace(regExp, replacer);\r\n  };\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_replacer.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_same-value.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_same-value.js ***!
  \*****************************************************/
/***/ (function(module) {

eval("// 7.2.9 SameValue(x, y)\r\nmodule.exports = Object.is || function is(x, y) {\r\n  // eslint-disable-next-line no-self-compare\r\n  return x === y ? x !== 0 || 1 / x === 1 / y : x != x && y != y;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_same-value.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_set-collection-from.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/_set-collection-from.js ***!
  \**************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// https://tc39.github.io/proposal-setmap-offrom/\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar aFunction = __webpack_require__(/*! ./_a-function */ \"./node_modules/core-js/modules/_a-function.js\");\r\nvar ctx = __webpack_require__(/*! ./_ctx */ \"./node_modules/core-js/modules/_ctx.js\");\r\nvar forOf = __webpack_require__(/*! ./_for-of */ \"./node_modules/core-js/modules/_for-of.js\");\r\n\r\nmodule.exports = function (COLLECTION) {\r\n  $export($export.S, COLLECTION, { from: function from(source /* , mapFn, thisArg */) {\r\n    var mapFn = arguments[1];\r\n    var mapping, A, n, cb;\r\n    aFunction(this);\r\n    mapping = mapFn !== undefined;\r\n    if (mapping) aFunction(mapFn);\r\n    if (source == undefined) return new this();\r\n    A = [];\r\n    if (mapping) {\r\n      n = 0;\r\n      cb = ctx(mapFn, arguments[2], 2);\r\n      forOf(source, false, function (nextItem) {\r\n        A.push(cb(nextItem, n++));\r\n      });\r\n    } else {\r\n      forOf(source, false, A.push, A);\r\n    }\r\n    return new this(A);\r\n  } });\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_set-collection-from.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_set-collection-of.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/_set-collection-of.js ***!
  \************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// https://tc39.github.io/proposal-setmap-offrom/\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\nmodule.exports = function (COLLECTION) {\r\n  $export($export.S, COLLECTION, { of: function of() {\r\n    var length = arguments.length;\r\n    var A = new Array(length);\r\n    while (length--) A[length] = arguments[length];\r\n    return new this(A);\r\n  } });\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_set-collection-of.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_set-proto.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/_set-proto.js ***!
  \****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// Works with __proto__ only. Old v8 can't work with null proto objects.\r\n/* eslint-disable no-proto */\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar check = function (O, proto) {\r\n  anObject(O);\r\n  if (!isObject(proto) && proto !== null) throw TypeError(proto + \": can't set as prototype!\");\r\n};\r\nmodule.exports = {\r\n  set: Object.setPrototypeOf || ('__proto__' in {} ? // eslint-disable-line\r\n    function (test, buggy, set) {\r\n      try {\r\n        set = __webpack_require__(/*! ./_ctx */ \"./node_modules/core-js/modules/_ctx.js\")(Function.call, __webpack_require__(/*! ./_object-gopd */ \"./node_modules/core-js/modules/_object-gopd.js\").f(Object.prototype, '__proto__').set, 2);\r\n        set(test, []);\r\n        buggy = !(test instanceof Array);\r\n      } catch (e) { buggy = true; }\r\n      return function setPrototypeOf(O, proto) {\r\n        check(O, proto);\r\n        if (buggy) O.__proto__ = proto;\r\n        else set(O, proto);\r\n        return O;\r\n      };\r\n    }({}, false) : undefined),\r\n  check: check\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_set-proto.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_set-species.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/_set-species.js ***!
  \******************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar dP = __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\");\r\nvar DESCRIPTORS = __webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\");\r\nvar SPECIES = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('species');\r\n\r\nmodule.exports = function (KEY) {\r\n  var C = global[KEY];\r\n  if (DESCRIPTORS && C && !C[SPECIES]) dP.f(C, SPECIES, {\r\n    configurable: true,\r\n    get: function () { return this; }\r\n  });\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_set-species.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_set-to-string-tag.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/_set-to-string-tag.js ***!
  \************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var def = __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\").f;\r\nvar has = __webpack_require__(/*! ./_has */ \"./node_modules/core-js/modules/_has.js\");\r\nvar TAG = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('toStringTag');\r\n\r\nmodule.exports = function (it, tag, stat) {\r\n  if (it && !has(it = stat ? it : it.prototype, TAG)) def(it, TAG, { configurable: true, value: tag });\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_set-to-string-tag.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_shared-key.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_shared-key.js ***!
  \*****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var shared = __webpack_require__(/*! ./_shared */ \"./node_modules/core-js/modules/_shared.js\")('keys');\r\nvar uid = __webpack_require__(/*! ./_uid */ \"./node_modules/core-js/modules/_uid.js\");\r\nmodule.exports = function (key) {\r\n  return shared[key] || (shared[key] = uid(key));\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_shared-key.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_shared.js":
/*!*************************************************!*\
  !*** ./node_modules/core-js/modules/_shared.js ***!
  \*************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var core = __webpack_require__(/*! ./_core */ \"./node_modules/core-js/modules/_core.js\");\r\nvar global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar SHARED = '__core-js_shared__';\r\nvar store = global[SHARED] || (global[SHARED] = {});\r\n\r\n(module.exports = function (key, value) {\r\n  return store[key] || (store[key] = value !== undefined ? value : {});\r\n})('versions', []).push({\r\n  version: core.version,\r\n  mode: __webpack_require__(/*! ./_library */ \"./node_modules/core-js/modules/_library.js\") ? 'pure' : 'global',\r\n  copyright: '© 2020 Denis Pushkarev (zloirock.ru)'\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_shared.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_species-constructor.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/_species-constructor.js ***!
  \**************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// 7.3.20 SpeciesConstructor(O, defaultConstructor)\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar aFunction = __webpack_require__(/*! ./_a-function */ \"./node_modules/core-js/modules/_a-function.js\");\r\nvar SPECIES = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('species');\r\nmodule.exports = function (O, D) {\r\n  var C = anObject(O).constructor;\r\n  var S;\r\n  return C === undefined || (S = anObject(C)[SPECIES]) == undefined ? D : aFunction(S);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_species-constructor.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_strict-method.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/_strict-method.js ***!
  \********************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar fails = __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\");\r\n\r\nmodule.exports = function (method, arg) {\r\n  return !!method && fails(function () {\r\n    // eslint-disable-next-line no-useless-call\r\n    arg ? method.call(null, function () { /* empty */ }, 1) : method.call(null);\r\n  });\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_strict-method.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_string-at.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/_string-at.js ***!
  \****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var toInteger = __webpack_require__(/*! ./_to-integer */ \"./node_modules/core-js/modules/_to-integer.js\");\r\nvar defined = __webpack_require__(/*! ./_defined */ \"./node_modules/core-js/modules/_defined.js\");\r\n// true  -> String#at\r\n// false -> String#codePointAt\r\nmodule.exports = function (TO_STRING) {\r\n  return function (that, pos) {\r\n    var s = String(defined(that));\r\n    var i = toInteger(pos);\r\n    var l = s.length;\r\n    var a, b;\r\n    if (i < 0 || i >= l) return TO_STRING ? '' : undefined;\r\n    a = s.charCodeAt(i);\r\n    return a < 0xd800 || a > 0xdbff || i + 1 === l || (b = s.charCodeAt(i + 1)) < 0xdc00 || b > 0xdfff\r\n      ? TO_STRING ? s.charAt(i) : a\r\n      : TO_STRING ? s.slice(i, i + 2) : (a - 0xd800 << 10) + (b - 0xdc00) + 0x10000;\r\n  };\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_string-at.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_string-context.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/_string-context.js ***!
  \*********************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// helper for String#{startsWith, endsWith, includes}\r\nvar isRegExp = __webpack_require__(/*! ./_is-regexp */ \"./node_modules/core-js/modules/_is-regexp.js\");\r\nvar defined = __webpack_require__(/*! ./_defined */ \"./node_modules/core-js/modules/_defined.js\");\r\n\r\nmodule.exports = function (that, searchString, NAME) {\r\n  if (isRegExp(searchString)) throw TypeError('String#' + NAME + \" doesn't accept regex!\");\r\n  return String(defined(that));\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_string-context.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_string-html.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/_string-html.js ***!
  \******************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar fails = __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\");\r\nvar defined = __webpack_require__(/*! ./_defined */ \"./node_modules/core-js/modules/_defined.js\");\r\nvar quot = /\"/g;\r\n// B.2.3.2.1 CreateHTML(string, tag, attribute, value)\r\nvar createHTML = function (string, tag, attribute, value) {\r\n  var S = String(defined(string));\r\n  var p1 = '<' + tag;\r\n  if (attribute !== '') p1 += ' ' + attribute + '=\"' + String(value).replace(quot, '&quot;') + '\"';\r\n  return p1 + '>' + S + '</' + tag + '>';\r\n};\r\nmodule.exports = function (NAME, exec) {\r\n  var O = {};\r\n  O[NAME] = exec(createHTML);\r\n  $export($export.P + $export.F * fails(function () {\r\n    var test = ''[NAME]('\"');\r\n    return test !== test.toLowerCase() || test.split('\"').length > 3;\r\n  }), 'String', O);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_string-html.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_string-pad.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_string-pad.js ***!
  \*****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// https://github.com/tc39/proposal-string-pad-start-end\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nvar repeat = __webpack_require__(/*! ./_string-repeat */ \"./node_modules/core-js/modules/_string-repeat.js\");\r\nvar defined = __webpack_require__(/*! ./_defined */ \"./node_modules/core-js/modules/_defined.js\");\r\n\r\nmodule.exports = function (that, maxLength, fillString, left) {\r\n  var S = String(defined(that));\r\n  var stringLength = S.length;\r\n  var fillStr = fillString === undefined ? ' ' : String(fillString);\r\n  var intMaxLength = toLength(maxLength);\r\n  if (intMaxLength <= stringLength || fillStr == '') return S;\r\n  var fillLen = intMaxLength - stringLength;\r\n  var stringFiller = repeat.call(fillStr, Math.ceil(fillLen / fillStr.length));\r\n  if (stringFiller.length > fillLen) stringFiller = stringFiller.slice(0, fillLen);\r\n  return left ? stringFiller + S : S + stringFiller;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_string-pad.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_string-repeat.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/_string-repeat.js ***!
  \********************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar toInteger = __webpack_require__(/*! ./_to-integer */ \"./node_modules/core-js/modules/_to-integer.js\");\r\nvar defined = __webpack_require__(/*! ./_defined */ \"./node_modules/core-js/modules/_defined.js\");\r\n\r\nmodule.exports = function repeat(count) {\r\n  var str = String(defined(this));\r\n  var res = '';\r\n  var n = toInteger(count);\r\n  if (n < 0 || n == Infinity) throw RangeError(\"Count can't be negative\");\r\n  for (;n > 0; (n >>>= 1) && (str += str)) if (n & 1) res += str;\r\n  return res;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_string-repeat.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_string-trim.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/_string-trim.js ***!
  \******************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar defined = __webpack_require__(/*! ./_defined */ \"./node_modules/core-js/modules/_defined.js\");\r\nvar fails = __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\");\r\nvar spaces = __webpack_require__(/*! ./_string-ws */ \"./node_modules/core-js/modules/_string-ws.js\");\r\nvar space = '[' + spaces + ']';\r\nvar non = '\\u200b\\u0085';\r\nvar ltrim = RegExp('^' + space + space + '*');\r\nvar rtrim = RegExp(space + space + '*$');\r\n\r\nvar exporter = function (KEY, exec, ALIAS) {\r\n  var exp = {};\r\n  var FORCE = fails(function () {\r\n    return !!spaces[KEY]() || non[KEY]() != non;\r\n  });\r\n  var fn = exp[KEY] = FORCE ? exec(trim) : spaces[KEY];\r\n  if (ALIAS) exp[ALIAS] = fn;\r\n  $export($export.P + $export.F * FORCE, 'String', exp);\r\n};\r\n\r\n// 1 -> String#trimLeft\r\n// 2 -> String#trimRight\r\n// 3 -> String#trim\r\nvar trim = exporter.trim = function (string, TYPE) {\r\n  string = String(defined(string));\r\n  if (TYPE & 1) string = string.replace(ltrim, '');\r\n  if (TYPE & 2) string = string.replace(rtrim, '');\r\n  return string;\r\n};\r\n\r\nmodule.exports = exporter;\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_string-trim.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_string-ws.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/_string-ws.js ***!
  \****************************************************/
/***/ (function(module) {

eval("module.exports = '\\x09\\x0A\\x0B\\x0C\\x0D\\x20\\xA0\\u1680\\u180E\\u2000\\u2001\\u2002\\u2003' +\r\n  '\\u2004\\u2005\\u2006\\u2007\\u2008\\u2009\\u200A\\u202F\\u205F\\u3000\\u2028\\u2029\\uFEFF';\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_string-ws.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_task.js":
/*!***********************************************!*\
  !*** ./node_modules/core-js/modules/_task.js ***!
  \***********************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var ctx = __webpack_require__(/*! ./_ctx */ \"./node_modules/core-js/modules/_ctx.js\");\r\nvar invoke = __webpack_require__(/*! ./_invoke */ \"./node_modules/core-js/modules/_invoke.js\");\r\nvar html = __webpack_require__(/*! ./_html */ \"./node_modules/core-js/modules/_html.js\");\r\nvar cel = __webpack_require__(/*! ./_dom-create */ \"./node_modules/core-js/modules/_dom-create.js\");\r\nvar global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar process = global.process;\r\nvar setTask = global.setImmediate;\r\nvar clearTask = global.clearImmediate;\r\nvar MessageChannel = global.MessageChannel;\r\nvar Dispatch = global.Dispatch;\r\nvar counter = 0;\r\nvar queue = {};\r\nvar ONREADYSTATECHANGE = 'onreadystatechange';\r\nvar defer, channel, port;\r\nvar run = function () {\r\n  var id = +this;\r\n  // eslint-disable-next-line no-prototype-builtins\r\n  if (queue.hasOwnProperty(id)) {\r\n    var fn = queue[id];\r\n    delete queue[id];\r\n    fn();\r\n  }\r\n};\r\nvar listener = function (event) {\r\n  run.call(event.data);\r\n};\r\n// Node.js 0.9+ & IE10+ has setImmediate, otherwise:\r\nif (!setTask || !clearTask) {\r\n  setTask = function setImmediate(fn) {\r\n    var args = [];\r\n    var i = 1;\r\n    while (arguments.length > i) args.push(arguments[i++]);\r\n    queue[++counter] = function () {\r\n      // eslint-disable-next-line no-new-func\r\n      invoke(typeof fn == 'function' ? fn : Function(fn), args);\r\n    };\r\n    defer(counter);\r\n    return counter;\r\n  };\r\n  clearTask = function clearImmediate(id) {\r\n    delete queue[id];\r\n  };\r\n  // Node.js 0.8-\r\n  if (__webpack_require__(/*! ./_cof */ \"./node_modules/core-js/modules/_cof.js\")(process) == 'process') {\r\n    defer = function (id) {\r\n      process.nextTick(ctx(run, id, 1));\r\n    };\r\n  // Sphere (JS game engine) Dispatch API\r\n  } else if (Dispatch && Dispatch.now) {\r\n    defer = function (id) {\r\n      Dispatch.now(ctx(run, id, 1));\r\n    };\r\n  // Browsers with MessageChannel, includes WebWorkers\r\n  } else if (MessageChannel) {\r\n    channel = new MessageChannel();\r\n    port = channel.port2;\r\n    channel.port1.onmessage = listener;\r\n    defer = ctx(port.postMessage, port, 1);\r\n  // Browsers with postMessage, skip WebWorkers\r\n  // IE8 has postMessage, but it's sync & typeof its postMessage is 'object'\r\n  } else if (global.addEventListener && typeof postMessage == 'function' && !global.importScripts) {\r\n    defer = function (id) {\r\n      global.postMessage(id + '', '*');\r\n    };\r\n    global.addEventListener('message', listener, false);\r\n  // IE8-\r\n  } else if (ONREADYSTATECHANGE in cel('script')) {\r\n    defer = function (id) {\r\n      html.appendChild(cel('script'))[ONREADYSTATECHANGE] = function () {\r\n        html.removeChild(this);\r\n        run.call(id);\r\n      };\r\n    };\r\n  // Rest old browsers\r\n  } else {\r\n    defer = function (id) {\r\n      setTimeout(ctx(run, id, 1), 0);\r\n    };\r\n  }\r\n}\r\nmodule.exports = {\r\n  set: setTask,\r\n  clear: clearTask\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_task.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_to-absolute-index.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/_to-absolute-index.js ***!
  \************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var toInteger = __webpack_require__(/*! ./_to-integer */ \"./node_modules/core-js/modules/_to-integer.js\");\r\nvar max = Math.max;\r\nvar min = Math.min;\r\nmodule.exports = function (index, length) {\r\n  index = toInteger(index);\r\n  return index < 0 ? max(index + length, 0) : min(index, length);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_to-absolute-index.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_to-index.js":
/*!***************************************************!*\
  !*** ./node_modules/core-js/modules/_to-index.js ***!
  \***************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// https://tc39.github.io/ecma262/#sec-toindex\r\nvar toInteger = __webpack_require__(/*! ./_to-integer */ \"./node_modules/core-js/modules/_to-integer.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nmodule.exports = function (it) {\r\n  if (it === undefined) return 0;\r\n  var number = toInteger(it);\r\n  var length = toLength(number);\r\n  if (number !== length) throw RangeError('Wrong length!');\r\n  return length;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_to-index.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_to-integer.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_to-integer.js ***!
  \*****************************************************/
/***/ (function(module) {

eval("// 7.1.4 ToInteger\r\nvar ceil = Math.ceil;\r\nvar floor = Math.floor;\r\nmodule.exports = function (it) {\r\n  return isNaN(it = +it) ? 0 : (it > 0 ? floor : ceil)(it);\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_to-integer.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_to-iobject.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_to-iobject.js ***!
  \*****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// to indexed object, toObject with fallback for non-array-like ES3 strings\r\nvar IObject = __webpack_require__(/*! ./_iobject */ \"./node_modules/core-js/modules/_iobject.js\");\r\nvar defined = __webpack_require__(/*! ./_defined */ \"./node_modules/core-js/modules/_defined.js\");\r\nmodule.exports = function (it) {\r\n  return IObject(defined(it));\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_to-iobject.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_to-length.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/_to-length.js ***!
  \****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// 7.1.15 ToLength\r\nvar toInteger = __webpack_require__(/*! ./_to-integer */ \"./node_modules/core-js/modules/_to-integer.js\");\r\nvar min = Math.min;\r\nmodule.exports = function (it) {\r\n  return it > 0 ? min(toInteger(it), 0x1fffffffffffff) : 0; // pow(2, 53) - 1 == 9007199254740991\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_to-length.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_to-object.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/_to-object.js ***!
  \****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// 7.1.13 ToObject(argument)\r\nvar defined = __webpack_require__(/*! ./_defined */ \"./node_modules/core-js/modules/_defined.js\");\r\nmodule.exports = function (it) {\r\n  return Object(defined(it));\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_to-object.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_to-primitive.js":
/*!*******************************************************!*\
  !*** ./node_modules/core-js/modules/_to-primitive.js ***!
  \*******************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("// 7.1.1 ToPrimitive(input [, PreferredType])\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\n// instead of the ES6 spec version, we didn't implement @@toPrimitive case\r\n// and the second argument - flag - preferred type is a string\r\nmodule.exports = function (it, S) {\r\n  if (!isObject(it)) return it;\r\n  var fn, val;\r\n  if (S && typeof (fn = it.toString) == 'function' && !isObject(val = fn.call(it))) return val;\r\n  if (typeof (fn = it.valueOf) == 'function' && !isObject(val = fn.call(it))) return val;\r\n  if (!S && typeof (fn = it.toString) == 'function' && !isObject(val = fn.call(it))) return val;\r\n  throw TypeError(\"Can't convert object to primitive value\");\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_to-primitive.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_typed-array.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/_typed-array.js ***!
  \******************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nif (__webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\")) {\r\n  var LIBRARY = __webpack_require__(/*! ./_library */ \"./node_modules/core-js/modules/_library.js\");\r\n  var global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\n  var fails = __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\");\r\n  var $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n  var $typed = __webpack_require__(/*! ./_typed */ \"./node_modules/core-js/modules/_typed.js\");\r\n  var $buffer = __webpack_require__(/*! ./_typed-buffer */ \"./node_modules/core-js/modules/_typed-buffer.js\");\r\n  var ctx = __webpack_require__(/*! ./_ctx */ \"./node_modules/core-js/modules/_ctx.js\");\r\n  var anInstance = __webpack_require__(/*! ./_an-instance */ \"./node_modules/core-js/modules/_an-instance.js\");\r\n  var propertyDesc = __webpack_require__(/*! ./_property-desc */ \"./node_modules/core-js/modules/_property-desc.js\");\r\n  var hide = __webpack_require__(/*! ./_hide */ \"./node_modules/core-js/modules/_hide.js\");\r\n  var redefineAll = __webpack_require__(/*! ./_redefine-all */ \"./node_modules/core-js/modules/_redefine-all.js\");\r\n  var toInteger = __webpack_require__(/*! ./_to-integer */ \"./node_modules/core-js/modules/_to-integer.js\");\r\n  var toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\n  var toIndex = __webpack_require__(/*! ./_to-index */ \"./node_modules/core-js/modules/_to-index.js\");\r\n  var toAbsoluteIndex = __webpack_require__(/*! ./_to-absolute-index */ \"./node_modules/core-js/modules/_to-absolute-index.js\");\r\n  var toPrimitive = __webpack_require__(/*! ./_to-primitive */ \"./node_modules/core-js/modules/_to-primitive.js\");\r\n  var has = __webpack_require__(/*! ./_has */ \"./node_modules/core-js/modules/_has.js\");\r\n  var classof = __webpack_require__(/*! ./_classof */ \"./node_modules/core-js/modules/_classof.js\");\r\n  var isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\n  var toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\n  var isArrayIter = __webpack_require__(/*! ./_is-array-iter */ \"./node_modules/core-js/modules/_is-array-iter.js\");\r\n  var create = __webpack_require__(/*! ./_object-create */ \"./node_modules/core-js/modules/_object-create.js\");\r\n  var getPrototypeOf = __webpack_require__(/*! ./_object-gpo */ \"./node_modules/core-js/modules/_object-gpo.js\");\r\n  var gOPN = __webpack_require__(/*! ./_object-gopn */ \"./node_modules/core-js/modules/_object-gopn.js\").f;\r\n  var getIterFn = __webpack_require__(/*! ./core.get-iterator-method */ \"./node_modules/core-js/modules/core.get-iterator-method.js\");\r\n  var uid = __webpack_require__(/*! ./_uid */ \"./node_modules/core-js/modules/_uid.js\");\r\n  var wks = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\");\r\n  var createArrayMethod = __webpack_require__(/*! ./_array-methods */ \"./node_modules/core-js/modules/_array-methods.js\");\r\n  var createArrayIncludes = __webpack_require__(/*! ./_array-includes */ \"./node_modules/core-js/modules/_array-includes.js\");\r\n  var speciesConstructor = __webpack_require__(/*! ./_species-constructor */ \"./node_modules/core-js/modules/_species-constructor.js\");\r\n  var ArrayIterators = __webpack_require__(/*! ./es6.array.iterator */ \"./node_modules/core-js/modules/es6.array.iterator.js\");\r\n  var Iterators = __webpack_require__(/*! ./_iterators */ \"./node_modules/core-js/modules/_iterators.js\");\r\n  var $iterDetect = __webpack_require__(/*! ./_iter-detect */ \"./node_modules/core-js/modules/_iter-detect.js\");\r\n  var setSpecies = __webpack_require__(/*! ./_set-species */ \"./node_modules/core-js/modules/_set-species.js\");\r\n  var arrayFill = __webpack_require__(/*! ./_array-fill */ \"./node_modules/core-js/modules/_array-fill.js\");\r\n  var arrayCopyWithin = __webpack_require__(/*! ./_array-copy-within */ \"./node_modules/core-js/modules/_array-copy-within.js\");\r\n  var $DP = __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\");\r\n  var $GOPD = __webpack_require__(/*! ./_object-gopd */ \"./node_modules/core-js/modules/_object-gopd.js\");\r\n  var dP = $DP.f;\r\n  var gOPD = $GOPD.f;\r\n  var RangeError = global.RangeError;\r\n  var TypeError = global.TypeError;\r\n  var Uint8Array = global.Uint8Array;\r\n  var ARRAY_BUFFER = 'ArrayBuffer';\r\n  var SHARED_BUFFER = 'Shared' + ARRAY_BUFFER;\r\n  var BYTES_PER_ELEMENT = 'BYTES_PER_ELEMENT';\r\n  var PROTOTYPE = 'prototype';\r\n  var ArrayProto = Array[PROTOTYPE];\r\n  var $ArrayBuffer = $buffer.ArrayBuffer;\r\n  var $DataView = $buffer.DataView;\r\n  var arrayForEach = createArrayMethod(0);\r\n  var arrayFilter = createArrayMethod(2);\r\n  var arraySome = createArrayMethod(3);\r\n  var arrayEvery = createArrayMethod(4);\r\n  var arrayFind = createArrayMethod(5);\r\n  var arrayFindIndex = createArrayMethod(6);\r\n  var arrayIncludes = createArrayIncludes(true);\r\n  var arrayIndexOf = createArrayIncludes(false);\r\n  var arrayValues = ArrayIterators.values;\r\n  var arrayKeys = ArrayIterators.keys;\r\n  var arrayEntries = ArrayIterators.entries;\r\n  var arrayLastIndexOf = ArrayProto.lastIndexOf;\r\n  var arrayReduce = ArrayProto.reduce;\r\n  var arrayReduceRight = ArrayProto.reduceRight;\r\n  var arrayJoin = ArrayProto.join;\r\n  var arraySort = ArrayProto.sort;\r\n  var arraySlice = ArrayProto.slice;\r\n  var arrayToString = ArrayProto.toString;\r\n  var arrayToLocaleString = ArrayProto.toLocaleString;\r\n  var ITERATOR = wks('iterator');\r\n  var TAG = wks('toStringTag');\r\n  var TYPED_CONSTRUCTOR = uid('typed_constructor');\r\n  var DEF_CONSTRUCTOR = uid('def_constructor');\r\n  var ALL_CONSTRUCTORS = $typed.CONSTR;\r\n  var TYPED_ARRAY = $typed.TYPED;\r\n  var VIEW = $typed.VIEW;\r\n  var WRONG_LENGTH = 'Wrong length!';\r\n\r\n  var $map = createArrayMethod(1, function (O, length) {\r\n    return allocate(speciesConstructor(O, O[DEF_CONSTRUCTOR]), length);\r\n  });\r\n\r\n  var LITTLE_ENDIAN = fails(function () {\r\n    // eslint-disable-next-line no-undef\r\n    return new Uint8Array(new Uint16Array([1]).buffer)[0] === 1;\r\n  });\r\n\r\n  var FORCED_SET = !!Uint8Array && !!Uint8Array[PROTOTYPE].set && fails(function () {\r\n    new Uint8Array(1).set({});\r\n  });\r\n\r\n  var toOffset = function (it, BYTES) {\r\n    var offset = toInteger(it);\r\n    if (offset < 0 || offset % BYTES) throw RangeError('Wrong offset!');\r\n    return offset;\r\n  };\r\n\r\n  var validate = function (it) {\r\n    if (isObject(it) && TYPED_ARRAY in it) return it;\r\n    throw TypeError(it + ' is not a typed array!');\r\n  };\r\n\r\n  var allocate = function (C, length) {\r\n    if (!(isObject(C) && TYPED_CONSTRUCTOR in C)) {\r\n      throw TypeError('It is not a typed array constructor!');\r\n    } return new C(length);\r\n  };\r\n\r\n  var speciesFromList = function (O, list) {\r\n    return fromList(speciesConstructor(O, O[DEF_CONSTRUCTOR]), list);\r\n  };\r\n\r\n  var fromList = function (C, list) {\r\n    var index = 0;\r\n    var length = list.length;\r\n    var result = allocate(C, length);\r\n    while (length > index) result[index] = list[index++];\r\n    return result;\r\n  };\r\n\r\n  var addGetter = function (it, key, internal) {\r\n    dP(it, key, { get: function () { return this._d[internal]; } });\r\n  };\r\n\r\n  var $from = function from(source /* , mapfn, thisArg */) {\r\n    var O = toObject(source);\r\n    var aLen = arguments.length;\r\n    var mapfn = aLen > 1 ? arguments[1] : undefined;\r\n    var mapping = mapfn !== undefined;\r\n    var iterFn = getIterFn(O);\r\n    var i, length, values, result, step, iterator;\r\n    if (iterFn != undefined && !isArrayIter(iterFn)) {\r\n      for (iterator = iterFn.call(O), values = [], i = 0; !(step = iterator.next()).done; i++) {\r\n        values.push(step.value);\r\n      } O = values;\r\n    }\r\n    if (mapping && aLen > 2) mapfn = ctx(mapfn, arguments[2], 2);\r\n    for (i = 0, length = toLength(O.length), result = allocate(this, length); length > i; i++) {\r\n      result[i] = mapping ? mapfn(O[i], i) : O[i];\r\n    }\r\n    return result;\r\n  };\r\n\r\n  var $of = function of(/* ...items */) {\r\n    var index = 0;\r\n    var length = arguments.length;\r\n    var result = allocate(this, length);\r\n    while (length > index) result[index] = arguments[index++];\r\n    return result;\r\n  };\r\n\r\n  // iOS Safari 6.x fails here\r\n  var TO_LOCALE_BUG = !!Uint8Array && fails(function () { arrayToLocaleString.call(new Uint8Array(1)); });\r\n\r\n  var $toLocaleString = function toLocaleString() {\r\n    return arrayToLocaleString.apply(TO_LOCALE_BUG ? arraySlice.call(validate(this)) : validate(this), arguments);\r\n  };\r\n\r\n  var proto = {\r\n    copyWithin: function copyWithin(target, start /* , end */) {\r\n      return arrayCopyWithin.call(validate(this), target, start, arguments.length > 2 ? arguments[2] : undefined);\r\n    },\r\n    every: function every(callbackfn /* , thisArg */) {\r\n      return arrayEvery(validate(this), callbackfn, arguments.length > 1 ? arguments[1] : undefined);\r\n    },\r\n    fill: function fill(value /* , start, end */) { // eslint-disable-line no-unused-vars\r\n      return arrayFill.apply(validate(this), arguments);\r\n    },\r\n    filter: function filter(callbackfn /* , thisArg */) {\r\n      return speciesFromList(this, arrayFilter(validate(this), callbackfn,\r\n        arguments.length > 1 ? arguments[1] : undefined));\r\n    },\r\n    find: function find(predicate /* , thisArg */) {\r\n      return arrayFind(validate(this), predicate, arguments.length > 1 ? arguments[1] : undefined);\r\n    },\r\n    findIndex: function findIndex(predicate /* , thisArg */) {\r\n      return arrayFindIndex(validate(this), predicate, arguments.length > 1 ? arguments[1] : undefined);\r\n    },\r\n    forEach: function forEach(callbackfn /* , thisArg */) {\r\n      arrayForEach(validate(this), callbackfn, arguments.length > 1 ? arguments[1] : undefined);\r\n    },\r\n    indexOf: function indexOf(searchElement /* , fromIndex */) {\r\n      return arrayIndexOf(validate(this), searchElement, arguments.length > 1 ? arguments[1] : undefined);\r\n    },\r\n    includes: function includes(searchElement /* , fromIndex */) {\r\n      return arrayIncludes(validate(this), searchElement, arguments.length > 1 ? arguments[1] : undefined);\r\n    },\r\n    join: function join(separator) { // eslint-disable-line no-unused-vars\r\n      return arrayJoin.apply(validate(this), arguments);\r\n    },\r\n    lastIndexOf: function lastIndexOf(searchElement /* , fromIndex */) { // eslint-disable-line no-unused-vars\r\n      return arrayLastIndexOf.apply(validate(this), arguments);\r\n    },\r\n    map: function map(mapfn /* , thisArg */) {\r\n      return $map(validate(this), mapfn, arguments.length > 1 ? arguments[1] : undefined);\r\n    },\r\n    reduce: function reduce(callbackfn /* , initialValue */) { // eslint-disable-line no-unused-vars\r\n      return arrayReduce.apply(validate(this), arguments);\r\n    },\r\n    reduceRight: function reduceRight(callbackfn /* , initialValue */) { // eslint-disable-line no-unused-vars\r\n      return arrayReduceRight.apply(validate(this), arguments);\r\n    },\r\n    reverse: function reverse() {\r\n      var that = this;\r\n      var length = validate(that).length;\r\n      var middle = Math.floor(length / 2);\r\n      var index = 0;\r\n      var value;\r\n      while (index < middle) {\r\n        value = that[index];\r\n        that[index++] = that[--length];\r\n        that[length] = value;\r\n      } return that;\r\n    },\r\n    some: function some(callbackfn /* , thisArg */) {\r\n      return arraySome(validate(this), callbackfn, arguments.length > 1 ? arguments[1] : undefined);\r\n    },\r\n    sort: function sort(comparefn) {\r\n      return arraySort.call(validate(this), comparefn);\r\n    },\r\n    subarray: function subarray(begin, end) {\r\n      var O = validate(this);\r\n      var length = O.length;\r\n      var $begin = toAbsoluteIndex(begin, length);\r\n      return new (speciesConstructor(O, O[DEF_CONSTRUCTOR]))(\r\n        O.buffer,\r\n        O.byteOffset + $begin * O.BYTES_PER_ELEMENT,\r\n        toLength((end === undefined ? length : toAbsoluteIndex(end, length)) - $begin)\r\n      );\r\n    }\r\n  };\r\n\r\n  var $slice = function slice(start, end) {\r\n    return speciesFromList(this, arraySlice.call(validate(this), start, end));\r\n  };\r\n\r\n  var $set = function set(arrayLike /* , offset */) {\r\n    validate(this);\r\n    var offset = toOffset(arguments[1], 1);\r\n    var length = this.length;\r\n    var src = toObject(arrayLike);\r\n    var len = toLength(src.length);\r\n    var index = 0;\r\n    if (len + offset > length) throw RangeError(WRONG_LENGTH);\r\n    while (index < len) this[offset + index] = src[index++];\r\n  };\r\n\r\n  var $iterators = {\r\n    entries: function entries() {\r\n      return arrayEntries.call(validate(this));\r\n    },\r\n    keys: function keys() {\r\n      return arrayKeys.call(validate(this));\r\n    },\r\n    values: function values() {\r\n      return arrayValues.call(validate(this));\r\n    }\r\n  };\r\n\r\n  var isTAIndex = function (target, key) {\r\n    return isObject(target)\r\n      && target[TYPED_ARRAY]\r\n      && typeof key != 'symbol'\r\n      && key in target\r\n      && String(+key) == String(key);\r\n  };\r\n  var $getDesc = function getOwnPropertyDescriptor(target, key) {\r\n    return isTAIndex(target, key = toPrimitive(key, true))\r\n      ? propertyDesc(2, target[key])\r\n      : gOPD(target, key);\r\n  };\r\n  var $setDesc = function defineProperty(target, key, desc) {\r\n    if (isTAIndex(target, key = toPrimitive(key, true))\r\n      && isObject(desc)\r\n      && has(desc, 'value')\r\n      && !has(desc, 'get')\r\n      && !has(desc, 'set')\r\n      // TODO: add validation descriptor w/o calling accessors\r\n      && !desc.configurable\r\n      && (!has(desc, 'writable') || desc.writable)\r\n      && (!has(desc, 'enumerable') || desc.enumerable)\r\n    ) {\r\n      target[key] = desc.value;\r\n      return target;\r\n    } return dP(target, key, desc);\r\n  };\r\n\r\n  if (!ALL_CONSTRUCTORS) {\r\n    $GOPD.f = $getDesc;\r\n    $DP.f = $setDesc;\r\n  }\r\n\r\n  $export($export.S + $export.F * !ALL_CONSTRUCTORS, 'Object', {\r\n    getOwnPropertyDescriptor: $getDesc,\r\n    defineProperty: $setDesc\r\n  });\r\n\r\n  if (fails(function () { arrayToString.call({}); })) {\r\n    arrayToString = arrayToLocaleString = function toString() {\r\n      return arrayJoin.call(this);\r\n    };\r\n  }\r\n\r\n  var $TypedArrayPrototype$ = redefineAll({}, proto);\r\n  redefineAll($TypedArrayPrototype$, $iterators);\r\n  hide($TypedArrayPrototype$, ITERATOR, $iterators.values);\r\n  redefineAll($TypedArrayPrototype$, {\r\n    slice: $slice,\r\n    set: $set,\r\n    constructor: function () { /* noop */ },\r\n    toString: arrayToString,\r\n    toLocaleString: $toLocaleString\r\n  });\r\n  addGetter($TypedArrayPrototype$, 'buffer', 'b');\r\n  addGetter($TypedArrayPrototype$, 'byteOffset', 'o');\r\n  addGetter($TypedArrayPrototype$, 'byteLength', 'l');\r\n  addGetter($TypedArrayPrototype$, 'length', 'e');\r\n  dP($TypedArrayPrototype$, TAG, {\r\n    get: function () { return this[TYPED_ARRAY]; }\r\n  });\r\n\r\n  // eslint-disable-next-line max-statements\r\n  module.exports = function (KEY, BYTES, wrapper, CLAMPED) {\r\n    CLAMPED = !!CLAMPED;\r\n    var NAME = KEY + (CLAMPED ? 'Clamped' : '') + 'Array';\r\n    var GETTER = 'get' + KEY;\r\n    var SETTER = 'set' + KEY;\r\n    var TypedArray = global[NAME];\r\n    var Base = TypedArray || {};\r\n    var TAC = TypedArray && getPrototypeOf(TypedArray);\r\n    var FORCED = !TypedArray || !$typed.ABV;\r\n    var O = {};\r\n    var TypedArrayPrototype = TypedArray && TypedArray[PROTOTYPE];\r\n    var getter = function (that, index) {\r\n      var data = that._d;\r\n      return data.v[GETTER](index * BYTES + data.o, LITTLE_ENDIAN);\r\n    };\r\n    var setter = function (that, index, value) {\r\n      var data = that._d;\r\n      if (CLAMPED) value = (value = Math.round(value)) < 0 ? 0 : value > 0xff ? 0xff : value & 0xff;\r\n      data.v[SETTER](index * BYTES + data.o, value, LITTLE_ENDIAN);\r\n    };\r\n    var addElement = function (that, index) {\r\n      dP(that, index, {\r\n        get: function () {\r\n          return getter(this, index);\r\n        },\r\n        set: function (value) {\r\n          return setter(this, index, value);\r\n        },\r\n        enumerable: true\r\n      });\r\n    };\r\n    if (FORCED) {\r\n      TypedArray = wrapper(function (that, data, $offset, $length) {\r\n        anInstance(that, TypedArray, NAME, '_d');\r\n        var index = 0;\r\n        var offset = 0;\r\n        var buffer, byteLength, length, klass;\r\n        if (!isObject(data)) {\r\n          length = toIndex(data);\r\n          byteLength = length * BYTES;\r\n          buffer = new $ArrayBuffer(byteLength);\r\n        } else if (data instanceof $ArrayBuffer || (klass = classof(data)) == ARRAY_BUFFER || klass == SHARED_BUFFER) {\r\n          buffer = data;\r\n          offset = toOffset($offset, BYTES);\r\n          var $len = data.byteLength;\r\n          if ($length === undefined) {\r\n            if ($len % BYTES) throw RangeError(WRONG_LENGTH);\r\n            byteLength = $len - offset;\r\n            if (byteLength < 0) throw RangeError(WRONG_LENGTH);\r\n          } else {\r\n            byteLength = toLength($length) * BYTES;\r\n            if (byteLength + offset > $len) throw RangeError(WRONG_LENGTH);\r\n          }\r\n          length = byteLength / BYTES;\r\n        } else if (TYPED_ARRAY in data) {\r\n          return fromList(TypedArray, data);\r\n        } else {\r\n          return $from.call(TypedArray, data);\r\n        }\r\n        hide(that, '_d', {\r\n          b: buffer,\r\n          o: offset,\r\n          l: byteLength,\r\n          e: length,\r\n          v: new $DataView(buffer)\r\n        });\r\n        while (index < length) addElement(that, index++);\r\n      });\r\n      TypedArrayPrototype = TypedArray[PROTOTYPE] = create($TypedArrayPrototype$);\r\n      hide(TypedArrayPrototype, 'constructor', TypedArray);\r\n    } else if (!fails(function () {\r\n      TypedArray(1);\r\n    }) || !fails(function () {\r\n      new TypedArray(-1); // eslint-disable-line no-new\r\n    }) || !$iterDetect(function (iter) {\r\n      new TypedArray(); // eslint-disable-line no-new\r\n      new TypedArray(null); // eslint-disable-line no-new\r\n      new TypedArray(1.5); // eslint-disable-line no-new\r\n      new TypedArray(iter); // eslint-disable-line no-new\r\n    }, true)) {\r\n      TypedArray = wrapper(function (that, data, $offset, $length) {\r\n        anInstance(that, TypedArray, NAME);\r\n        var klass;\r\n        // `ws` module bug, temporarily remove validation length for Uint8Array\r\n        // https://github.com/websockets/ws/pull/645\r\n        if (!isObject(data)) return new Base(toIndex(data));\r\n        if (data instanceof $ArrayBuffer || (klass = classof(data)) == ARRAY_BUFFER || klass == SHARED_BUFFER) {\r\n          return $length !== undefined\r\n            ? new Base(data, toOffset($offset, BYTES), $length)\r\n            : $offset !== undefined\r\n              ? new Base(data, toOffset($offset, BYTES))\r\n              : new Base(data);\r\n        }\r\n        if (TYPED_ARRAY in data) return fromList(TypedArray, data);\r\n        return $from.call(TypedArray, data);\r\n      });\r\n      arrayForEach(TAC !== Function.prototype ? gOPN(Base).concat(gOPN(TAC)) : gOPN(Base), function (key) {\r\n        if (!(key in TypedArray)) hide(TypedArray, key, Base[key]);\r\n      });\r\n      TypedArray[PROTOTYPE] = TypedArrayPrototype;\r\n      if (!LIBRARY) TypedArrayPrototype.constructor = TypedArray;\r\n    }\r\n    var $nativeIterator = TypedArrayPrototype[ITERATOR];\r\n    var CORRECT_ITER_NAME = !!$nativeIterator\r\n      && ($nativeIterator.name == 'values' || $nativeIterator.name == undefined);\r\n    var $iterator = $iterators.values;\r\n    hide(TypedArray, TYPED_CONSTRUCTOR, true);\r\n    hide(TypedArrayPrototype, TYPED_ARRAY, NAME);\r\n    hide(TypedArrayPrototype, VIEW, true);\r\n    hide(TypedArrayPrototype, DEF_CONSTRUCTOR, TypedArray);\r\n\r\n    if (CLAMPED ? new TypedArray(1)[TAG] != NAME : !(TAG in TypedArrayPrototype)) {\r\n      dP(TypedArrayPrototype, TAG, {\r\n        get: function () { return NAME; }\r\n      });\r\n    }\r\n\r\n    O[NAME] = TypedArray;\r\n\r\n    $export($export.G + $export.W + $export.F * (TypedArray != Base), O);\r\n\r\n    $export($export.S, NAME, {\r\n      BYTES_PER_ELEMENT: BYTES\r\n    });\r\n\r\n    $export($export.S + $export.F * fails(function () { Base.of.call(TypedArray, 1); }), NAME, {\r\n      from: $from,\r\n      of: $of\r\n    });\r\n\r\n    if (!(BYTES_PER_ELEMENT in TypedArrayPrototype)) hide(TypedArrayPrototype, BYTES_PER_ELEMENT, BYTES);\r\n\r\n    $export($export.P, NAME, proto);\r\n\r\n    setSpecies(NAME);\r\n\r\n    $export($export.P + $export.F * FORCED_SET, NAME, { set: $set });\r\n\r\n    $export($export.P + $export.F * !CORRECT_ITER_NAME, NAME, $iterators);\r\n\r\n    if (!LIBRARY && TypedArrayPrototype.toString != arrayToString) TypedArrayPrototype.toString = arrayToString;\r\n\r\n    $export($export.P + $export.F * fails(function () {\r\n      new TypedArray(1).slice();\r\n    }), NAME, { slice: $slice });\r\n\r\n    $export($export.P + $export.F * (fails(function () {\r\n      return [1, 2].toLocaleString() != new TypedArray([1, 2]).toLocaleString();\r\n    }) || !fails(function () {\r\n      TypedArrayPrototype.toLocaleString.call([1, 2]);\r\n    })), NAME, { toLocaleString: $toLocaleString });\r\n\r\n    Iterators[NAME] = CORRECT_ITER_NAME ? $nativeIterator : $iterator;\r\n    if (!LIBRARY && !CORRECT_ITER_NAME) hide(TypedArrayPrototype, ITERATOR, $iterator);\r\n  };\r\n} else module.exports = function () { /* empty */ };\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_typed-array.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_typed-buffer.js":
/*!*******************************************************!*\
  !*** ./node_modules/core-js/modules/_typed-buffer.js ***!
  \*******************************************************/
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {

"use strict";
eval("\r\nvar global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar DESCRIPTORS = __webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\");\r\nvar LIBRARY = __webpack_require__(/*! ./_library */ \"./node_modules/core-js/modules/_library.js\");\r\nvar $typed = __webpack_require__(/*! ./_typed */ \"./node_modules/core-js/modules/_typed.js\");\r\nvar hide = __webpack_require__(/*! ./_hide */ \"./node_modules/core-js/modules/_hide.js\");\r\nvar redefineAll = __webpack_require__(/*! ./_redefine-all */ \"./node_modules/core-js/modules/_redefine-all.js\");\r\nvar fails = __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\");\r\nvar anInstance = __webpack_require__(/*! ./_an-instance */ \"./node_modules/core-js/modules/_an-instance.js\");\r\nvar toInteger = __webpack_require__(/*! ./_to-integer */ \"./node_modules/core-js/modules/_to-integer.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nvar toIndex = __webpack_require__(/*! ./_to-index */ \"./node_modules/core-js/modules/_to-index.js\");\r\nvar gOPN = __webpack_require__(/*! ./_object-gopn */ \"./node_modules/core-js/modules/_object-gopn.js\").f;\r\nvar dP = __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\").f;\r\nvar arrayFill = __webpack_require__(/*! ./_array-fill */ \"./node_modules/core-js/modules/_array-fill.js\");\r\nvar setToStringTag = __webpack_require__(/*! ./_set-to-string-tag */ \"./node_modules/core-js/modules/_set-to-string-tag.js\");\r\nvar ARRAY_BUFFER = 'ArrayBuffer';\r\nvar DATA_VIEW = 'DataView';\r\nvar PROTOTYPE = 'prototype';\r\nvar WRONG_LENGTH = 'Wrong length!';\r\nvar WRONG_INDEX = 'Wrong index!';\r\nvar $ArrayBuffer = global[ARRAY_BUFFER];\r\nvar $DataView = global[DATA_VIEW];\r\nvar Math = global.Math;\r\nvar RangeError = global.RangeError;\r\n// eslint-disable-next-line no-shadow-restricted-names\r\nvar Infinity = global.Infinity;\r\nvar BaseBuffer = $ArrayBuffer;\r\nvar abs = Math.abs;\r\nvar pow = Math.pow;\r\nvar floor = Math.floor;\r\nvar log = Math.log;\r\nvar LN2 = Math.LN2;\r\nvar BUFFER = 'buffer';\r\nvar BYTE_LENGTH = 'byteLength';\r\nvar BYTE_OFFSET = 'byteOffset';\r\nvar $BUFFER = DESCRIPTORS ? '_b' : BUFFER;\r\nvar $LENGTH = DESCRIPTORS ? '_l' : BYTE_LENGTH;\r\nvar $OFFSET = DESCRIPTORS ? '_o' : BYTE_OFFSET;\r\n\r\n// IEEE754 conversions based on https://github.com/feross/ieee754\r\nfunction packIEEE754(value, mLen, nBytes) {\r\n  var buffer = new Array(nBytes);\r\n  var eLen = nBytes * 8 - mLen - 1;\r\n  var eMax = (1 << eLen) - 1;\r\n  var eBias = eMax >> 1;\r\n  var rt = mLen === 23 ? pow(2, -24) - pow(2, -77) : 0;\r\n  var i = 0;\r\n  var s = value < 0 || value === 0 && 1 / value < 0 ? 1 : 0;\r\n  var e, m, c;\r\n  value = abs(value);\r\n  // eslint-disable-next-line no-self-compare\r\n  if (value != value || value === Infinity) {\r\n    // eslint-disable-next-line no-self-compare\r\n    m = value != value ? 1 : 0;\r\n    e = eMax;\r\n  } else {\r\n    e = floor(log(value) / LN2);\r\n    if (value * (c = pow(2, -e)) < 1) {\r\n      e--;\r\n      c *= 2;\r\n    }\r\n    if (e + eBias >= 1) {\r\n      value += rt / c;\r\n    } else {\r\n      value += rt * pow(2, 1 - eBias);\r\n    }\r\n    if (value * c >= 2) {\r\n      e++;\r\n      c /= 2;\r\n    }\r\n    if (e + eBias >= eMax) {\r\n      m = 0;\r\n      e = eMax;\r\n    } else if (e + eBias >= 1) {\r\n      m = (value * c - 1) * pow(2, mLen);\r\n      e = e + eBias;\r\n    } else {\r\n      m = value * pow(2, eBias - 1) * pow(2, mLen);\r\n      e = 0;\r\n    }\r\n  }\r\n  for (; mLen >= 8; buffer[i++] = m & 255, m /= 256, mLen -= 8);\r\n  e = e << mLen | m;\r\n  eLen += mLen;\r\n  for (; eLen > 0; buffer[i++] = e & 255, e /= 256, eLen -= 8);\r\n  buffer[--i] |= s * 128;\r\n  return buffer;\r\n}\r\nfunction unpackIEEE754(buffer, mLen, nBytes) {\r\n  var eLen = nBytes * 8 - mLen - 1;\r\n  var eMax = (1 << eLen) - 1;\r\n  var eBias = eMax >> 1;\r\n  var nBits = eLen - 7;\r\n  var i = nBytes - 1;\r\n  var s = buffer[i--];\r\n  var e = s & 127;\r\n  var m;\r\n  s >>= 7;\r\n  for (; nBits > 0; e = e * 256 + buffer[i], i--, nBits -= 8);\r\n  m = e & (1 << -nBits) - 1;\r\n  e >>= -nBits;\r\n  nBits += mLen;\r\n  for (; nBits > 0; m = m * 256 + buffer[i], i--, nBits -= 8);\r\n  if (e === 0) {\r\n    e = 1 - eBias;\r\n  } else if (e === eMax) {\r\n    return m ? NaN : s ? -Infinity : Infinity;\r\n  } else {\r\n    m = m + pow(2, mLen);\r\n    e = e - eBias;\r\n  } return (s ? -1 : 1) * m * pow(2, e - mLen);\r\n}\r\n\r\nfunction unpackI32(bytes) {\r\n  return bytes[3] << 24 | bytes[2] << 16 | bytes[1] << 8 | bytes[0];\r\n}\r\nfunction packI8(it) {\r\n  return [it & 0xff];\r\n}\r\nfunction packI16(it) {\r\n  return [it & 0xff, it >> 8 & 0xff];\r\n}\r\nfunction packI32(it) {\r\n  return [it & 0xff, it >> 8 & 0xff, it >> 16 & 0xff, it >> 24 & 0xff];\r\n}\r\nfunction packF64(it) {\r\n  return packIEEE754(it, 52, 8);\r\n}\r\nfunction packF32(it) {\r\n  return packIEEE754(it, 23, 4);\r\n}\r\n\r\nfunction addGetter(C, key, internal) {\r\n  dP(C[PROTOTYPE], key, { get: function () { return this[internal]; } });\r\n}\r\n\r\nfunction get(view, bytes, index, isLittleEndian) {\r\n  var numIndex = +index;\r\n  var intIndex = toIndex(numIndex);\r\n  if (intIndex + bytes > view[$LENGTH]) throw RangeError(WRONG_INDEX);\r\n  var store = view[$BUFFER]._b;\r\n  var start = intIndex + view[$OFFSET];\r\n  var pack = store.slice(start, start + bytes);\r\n  return isLittleEndian ? pack : pack.reverse();\r\n}\r\nfunction set(view, bytes, index, conversion, value, isLittleEndian) {\r\n  var numIndex = +index;\r\n  var intIndex = toIndex(numIndex);\r\n  if (intIndex + bytes > view[$LENGTH]) throw RangeError(WRONG_INDEX);\r\n  var store = view[$BUFFER]._b;\r\n  var start = intIndex + view[$OFFSET];\r\n  var pack = conversion(+value);\r\n  for (var i = 0; i < bytes; i++) store[start + i] = pack[isLittleEndian ? i : bytes - i - 1];\r\n}\r\n\r\nif (!$typed.ABV) {\r\n  $ArrayBuffer = function ArrayBuffer(length) {\r\n    anInstance(this, $ArrayBuffer, ARRAY_BUFFER);\r\n    var byteLength = toIndex(length);\r\n    this._b = arrayFill.call(new Array(byteLength), 0);\r\n    this[$LENGTH] = byteLength;\r\n  };\r\n\r\n  $DataView = function DataView(buffer, byteOffset, byteLength) {\r\n    anInstance(this, $DataView, DATA_VIEW);\r\n    anInstance(buffer, $ArrayBuffer, DATA_VIEW);\r\n    var bufferLength = buffer[$LENGTH];\r\n    var offset = toInteger(byteOffset);\r\n    if (offset < 0 || offset > bufferLength) throw RangeError('Wrong offset!');\r\n    byteLength = byteLength === undefined ? bufferLength - offset : toLength(byteLength);\r\n    if (offset + byteLength > bufferLength) throw RangeError(WRONG_LENGTH);\r\n    this[$BUFFER] = buffer;\r\n    this[$OFFSET] = offset;\r\n    this[$LENGTH] = byteLength;\r\n  };\r\n\r\n  if (DESCRIPTORS) {\r\n    addGetter($ArrayBuffer, BYTE_LENGTH, '_l');\r\n    addGetter($DataView, BUFFER, '_b');\r\n    addGetter($DataView, BYTE_LENGTH, '_l');\r\n    addGetter($DataView, BYTE_OFFSET, '_o');\r\n  }\r\n\r\n  redefineAll($DataView[PROTOTYPE], {\r\n    getInt8: function getInt8(byteOffset) {\r\n      return get(this, 1, byteOffset)[0] << 24 >> 24;\r\n    },\r\n    getUint8: function getUint8(byteOffset) {\r\n      return get(this, 1, byteOffset)[0];\r\n    },\r\n    getInt16: function getInt16(byteOffset /* , littleEndian */) {\r\n      var bytes = get(this, 2, byteOffset, arguments[1]);\r\n      return (bytes[1] << 8 | bytes[0]) << 16 >> 16;\r\n    },\r\n    getUint16: function getUint16(byteOffset /* , littleEndian */) {\r\n      var bytes = get(this, 2, byteOffset, arguments[1]);\r\n      return bytes[1] << 8 | bytes[0];\r\n    },\r\n    getInt32: function getInt32(byteOffset /* , littleEndian */) {\r\n      return unpackI32(get(this, 4, byteOffset, arguments[1]));\r\n    },\r\n    getUint32: function getUint32(byteOffset /* , littleEndian */) {\r\n      return unpackI32(get(this, 4, byteOffset, arguments[1])) >>> 0;\r\n    },\r\n    getFloat32: function getFloat32(byteOffset /* , littleEndian */) {\r\n      return unpackIEEE754(get(this, 4, byteOffset, arguments[1]), 23, 4);\r\n    },\r\n    getFloat64: function getFloat64(byteOffset /* , littleEndian */) {\r\n      return unpackIEEE754(get(this, 8, byteOffset, arguments[1]), 52, 8);\r\n    },\r\n    setInt8: function setInt8(byteOffset, value) {\r\n      set(this, 1, byteOffset, packI8, value);\r\n    },\r\n    setUint8: function setUint8(byteOffset, value) {\r\n      set(this, 1, byteOffset, packI8, value);\r\n    },\r\n    setInt16: function setInt16(byteOffset, value /* , littleEndian */) {\r\n      set(this, 2, byteOffset, packI16, value, arguments[2]);\r\n    },\r\n    setUint16: function setUint16(byteOffset, value /* , littleEndian */) {\r\n      set(this, 2, byteOffset, packI16, value, arguments[2]);\r\n    },\r\n    setInt32: function setInt32(byteOffset, value /* , littleEndian */) {\r\n      set(this, 4, byteOffset, packI32, value, arguments[2]);\r\n    },\r\n    setUint32: function setUint32(byteOffset, value /* , littleEndian */) {\r\n      set(this, 4, byteOffset, packI32, value, arguments[2]);\r\n    },\r\n    setFloat32: function setFloat32(byteOffset, value /* , littleEndian */) {\r\n      set(this, 4, byteOffset, packF32, value, arguments[2]);\r\n    },\r\n    setFloat64: function setFloat64(byteOffset, value /* , littleEndian */) {\r\n      set(this, 8, byteOffset, packF64, value, arguments[2]);\r\n    }\r\n  });\r\n} else {\r\n  if (!fails(function () {\r\n    $ArrayBuffer(1);\r\n  }) || !fails(function () {\r\n    new $ArrayBuffer(-1); // eslint-disable-line no-new\r\n  }) || fails(function () {\r\n    new $ArrayBuffer(); // eslint-disable-line no-new\r\n    new $ArrayBuffer(1.5); // eslint-disable-line no-new\r\n    new $ArrayBuffer(NaN); // eslint-disable-line no-new\r\n    return $ArrayBuffer.name != ARRAY_BUFFER;\r\n  })) {\r\n    $ArrayBuffer = function ArrayBuffer(length) {\r\n      anInstance(this, $ArrayBuffer);\r\n      return new BaseBuffer(toIndex(length));\r\n    };\r\n    var ArrayBufferProto = $ArrayBuffer[PROTOTYPE] = BaseBuffer[PROTOTYPE];\r\n    for (var keys = gOPN(BaseBuffer), j = 0, key; keys.length > j;) {\r\n      if (!((key = keys[j++]) in $ArrayBuffer)) hide($ArrayBuffer, key, BaseBuffer[key]);\r\n    }\r\n    if (!LIBRARY) ArrayBufferProto.constructor = $ArrayBuffer;\r\n  }\r\n  // iOS Safari 7.x bug\r\n  var view = new $DataView(new $ArrayBuffer(2));\r\n  var $setInt8 = $DataView[PROTOTYPE].setInt8;\r\n  view.setInt8(0, 2147483648);\r\n  view.setInt8(1, 2147483649);\r\n  if (view.getInt8(0) || !view.getInt8(1)) redefineAll($DataView[PROTOTYPE], {\r\n    setInt8: function setInt8(byteOffset, value) {\r\n      $setInt8.call(this, byteOffset, value << 24 >> 24);\r\n    },\r\n    setUint8: function setUint8(byteOffset, value) {\r\n      $setInt8.call(this, byteOffset, value << 24 >> 24);\r\n    }\r\n  }, true);\r\n}\r\nsetToStringTag($ArrayBuffer, ARRAY_BUFFER);\r\nsetToStringTag($DataView, DATA_VIEW);\r\nhide($DataView[PROTOTYPE], $typed.VIEW, true);\r\nexports[ARRAY_BUFFER] = $ArrayBuffer;\r\nexports[DATA_VIEW] = $DataView;\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_typed-buffer.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_typed.js":
/*!************************************************!*\
  !*** ./node_modules/core-js/modules/_typed.js ***!
  \************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar hide = __webpack_require__(/*! ./_hide */ \"./node_modules/core-js/modules/_hide.js\");\r\nvar uid = __webpack_require__(/*! ./_uid */ \"./node_modules/core-js/modules/_uid.js\");\r\nvar TYPED = uid('typed_array');\r\nvar VIEW = uid('view');\r\nvar ABV = !!(global.ArrayBuffer && global.DataView);\r\nvar CONSTR = ABV;\r\nvar i = 0;\r\nvar l = 9;\r\nvar Typed;\r\n\r\nvar TypedArrayConstructors = (\r\n  'Int8Array,Uint8Array,Uint8ClampedArray,Int16Array,Uint16Array,Int32Array,Uint32Array,Float32Array,Float64Array'\r\n).split(',');\r\n\r\nwhile (i < l) {\r\n  if (Typed = global[TypedArrayConstructors[i++]]) {\r\n    hide(Typed.prototype, TYPED, true);\r\n    hide(Typed.prototype, VIEW, true);\r\n  } else CONSTR = false;\r\n}\r\n\r\nmodule.exports = {\r\n  ABV: ABV,\r\n  CONSTR: CONSTR,\r\n  TYPED: TYPED,\r\n  VIEW: VIEW\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_typed.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_uid.js":
/*!**********************************************!*\
  !*** ./node_modules/core-js/modules/_uid.js ***!
  \**********************************************/
/***/ (function(module) {

eval("var id = 0;\r\nvar px = Math.random();\r\nmodule.exports = function (key) {\r\n  return 'Symbol('.concat(key === undefined ? '' : key, ')_', (++id + px).toString(36));\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_uid.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_user-agent.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_user-agent.js ***!
  \*****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar navigator = global.navigator;\r\n\r\nmodule.exports = navigator && navigator.userAgent || '';\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_user-agent.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_validate-collection.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/_validate-collection.js ***!
  \**************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nmodule.exports = function (it, TYPE) {\r\n  if (!isObject(it) || it._t !== TYPE) throw TypeError('Incompatible receiver, ' + TYPE + ' required!');\r\n  return it;\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_validate-collection.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_wks-define.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/_wks-define.js ***!
  \*****************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar core = __webpack_require__(/*! ./_core */ \"./node_modules/core-js/modules/_core.js\");\r\nvar LIBRARY = __webpack_require__(/*! ./_library */ \"./node_modules/core-js/modules/_library.js\");\r\nvar wksExt = __webpack_require__(/*! ./_wks-ext */ \"./node_modules/core-js/modules/_wks-ext.js\");\r\nvar defineProperty = __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\").f;\r\nmodule.exports = function (name) {\r\n  var $Symbol = core.Symbol || (core.Symbol = LIBRARY ? {} : global.Symbol || {});\r\n  if (name.charAt(0) != '_' && !(name in $Symbol)) defineProperty($Symbol, name, { value: wksExt.f(name) });\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_wks-define.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_wks-ext.js":
/*!**************************************************!*\
  !*** ./node_modules/core-js/modules/_wks-ext.js ***!
  \**************************************************/
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {

eval("exports.f = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\");\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_wks-ext.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/_wks.js":
/*!**********************************************!*\
  !*** ./node_modules/core-js/modules/_wks.js ***!
  \**********************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var store = __webpack_require__(/*! ./_shared */ \"./node_modules/core-js/modules/_shared.js\")('wks');\r\nvar uid = __webpack_require__(/*! ./_uid */ \"./node_modules/core-js/modules/_uid.js\");\r\nvar Symbol = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\").Symbol;\r\nvar USE_SYMBOL = typeof Symbol == 'function';\r\n\r\nvar $exports = module.exports = function (name) {\r\n  return store[name] || (store[name] =\r\n    USE_SYMBOL && Symbol[name] || (USE_SYMBOL ? Symbol : uid)('Symbol.' + name));\r\n};\r\n\r\n$exports.store = store;\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/_wks.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/core.get-iterator-method.js":
/*!******************************************************************!*\
  !*** ./node_modules/core-js/modules/core.get-iterator-method.js ***!
  \******************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("var classof = __webpack_require__(/*! ./_classof */ \"./node_modules/core-js/modules/_classof.js\");\r\nvar ITERATOR = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('iterator');\r\nvar Iterators = __webpack_require__(/*! ./_iterators */ \"./node_modules/core-js/modules/_iterators.js\");\r\nmodule.exports = __webpack_require__(/*! ./_core */ \"./node_modules/core-js/modules/_core.js\").getIteratorMethod = function (it) {\r\n  if (it != undefined) return it[ITERATOR]\r\n    || it['@@iterator']\r\n    || Iterators[classof(it)];\r\n};\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/core.get-iterator-method.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/core.regexp.escape.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/core.regexp.escape.js ***!
  \************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://github.com/benjamingr/RexExp.escape\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $re = __webpack_require__(/*! ./_replacer */ \"./node_modules/core-js/modules/_replacer.js\")(/[\\\\^$*+?.()|[\\]{}]/g, '\\\\$&');\r\n\r\n$export($export.S, 'RegExp', { escape: function escape(it) { return $re(it); } });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/core.regexp.escape.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.copy-within.js":
/*!***************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.copy-within.js ***!
  \***************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 22.1.3.3 Array.prototype.copyWithin(target, start, end = this.length)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.P, 'Array', { copyWithin: __webpack_require__(/*! ./_array-copy-within */ \"./node_modules/core-js/modules/_array-copy-within.js\") });\r\n\r\n__webpack_require__(/*! ./_add-to-unscopables */ \"./node_modules/core-js/modules/_add-to-unscopables.js\")('copyWithin');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.copy-within.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.every.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.every.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $every = __webpack_require__(/*! ./_array-methods */ \"./node_modules/core-js/modules/_array-methods.js\")(4);\r\n\r\n$export($export.P + $export.F * !__webpack_require__(/*! ./_strict-method */ \"./node_modules/core-js/modules/_strict-method.js\")([].every, true), 'Array', {\r\n  // 22.1.3.5 / 15.4.4.16 Array.prototype.every(callbackfn [, thisArg])\r\n  every: function every(callbackfn /* , thisArg */) {\r\n    return $every(this, callbackfn, arguments[1]);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.every.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.fill.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.fill.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 22.1.3.6 Array.prototype.fill(value, start = 0, end = this.length)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.P, 'Array', { fill: __webpack_require__(/*! ./_array-fill */ \"./node_modules/core-js/modules/_array-fill.js\") });\r\n\r\n__webpack_require__(/*! ./_add-to-unscopables */ \"./node_modules/core-js/modules/_add-to-unscopables.js\")('fill');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.fill.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.filter.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.filter.js ***!
  \**********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $filter = __webpack_require__(/*! ./_array-methods */ \"./node_modules/core-js/modules/_array-methods.js\")(2);\r\n\r\n$export($export.P + $export.F * !__webpack_require__(/*! ./_strict-method */ \"./node_modules/core-js/modules/_strict-method.js\")([].filter, true), 'Array', {\r\n  // 22.1.3.7 / 15.4.4.20 Array.prototype.filter(callbackfn [, thisArg])\r\n  filter: function filter(callbackfn /* , thisArg */) {\r\n    return $filter(this, callbackfn, arguments[1]);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.filter.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.find-index.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.find-index.js ***!
  \**************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// 22.1.3.9 Array.prototype.findIndex(predicate, thisArg = undefined)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $find = __webpack_require__(/*! ./_array-methods */ \"./node_modules/core-js/modules/_array-methods.js\")(6);\r\nvar KEY = 'findIndex';\r\nvar forced = true;\r\n// Shouldn't skip holes\r\nif (KEY in []) Array(1)[KEY](function () { forced = false; });\r\n$export($export.P + $export.F * forced, 'Array', {\r\n  findIndex: function findIndex(callbackfn /* , that = undefined */) {\r\n    return $find(this, callbackfn, arguments.length > 1 ? arguments[1] : undefined);\r\n  }\r\n});\r\n__webpack_require__(/*! ./_add-to-unscopables */ \"./node_modules/core-js/modules/_add-to-unscopables.js\")(KEY);\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.find-index.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.find.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.find.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// 22.1.3.8 Array.prototype.find(predicate, thisArg = undefined)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $find = __webpack_require__(/*! ./_array-methods */ \"./node_modules/core-js/modules/_array-methods.js\")(5);\r\nvar KEY = 'find';\r\nvar forced = true;\r\n// Shouldn't skip holes\r\nif (KEY in []) Array(1)[KEY](function () { forced = false; });\r\n$export($export.P + $export.F * forced, 'Array', {\r\n  find: function find(callbackfn /* , that = undefined */) {\r\n    return $find(this, callbackfn, arguments.length > 1 ? arguments[1] : undefined);\r\n  }\r\n});\r\n__webpack_require__(/*! ./_add-to-unscopables */ \"./node_modules/core-js/modules/_add-to-unscopables.js\")(KEY);\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.find.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.for-each.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.for-each.js ***!
  \************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $forEach = __webpack_require__(/*! ./_array-methods */ \"./node_modules/core-js/modules/_array-methods.js\")(0);\r\nvar STRICT = __webpack_require__(/*! ./_strict-method */ \"./node_modules/core-js/modules/_strict-method.js\")([].forEach, true);\r\n\r\n$export($export.P + $export.F * !STRICT, 'Array', {\r\n  // 22.1.3.10 / 15.4.4.18 Array.prototype.forEach(callbackfn [, thisArg])\r\n  forEach: function forEach(callbackfn /* , thisArg */) {\r\n    return $forEach(this, callbackfn, arguments[1]);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.for-each.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.from.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.from.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar ctx = __webpack_require__(/*! ./_ctx */ \"./node_modules/core-js/modules/_ctx.js\");\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar call = __webpack_require__(/*! ./_iter-call */ \"./node_modules/core-js/modules/_iter-call.js\");\r\nvar isArrayIter = __webpack_require__(/*! ./_is-array-iter */ \"./node_modules/core-js/modules/_is-array-iter.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nvar createProperty = __webpack_require__(/*! ./_create-property */ \"./node_modules/core-js/modules/_create-property.js\");\r\nvar getIterFn = __webpack_require__(/*! ./core.get-iterator-method */ \"./node_modules/core-js/modules/core.get-iterator-method.js\");\r\n\r\n$export($export.S + $export.F * !__webpack_require__(/*! ./_iter-detect */ \"./node_modules/core-js/modules/_iter-detect.js\")(function (iter) { Array.from(iter); }), 'Array', {\r\n  // 22.1.2.1 Array.from(arrayLike, mapfn = undefined, thisArg = undefined)\r\n  from: function from(arrayLike /* , mapfn = undefined, thisArg = undefined */) {\r\n    var O = toObject(arrayLike);\r\n    var C = typeof this == 'function' ? this : Array;\r\n    var aLen = arguments.length;\r\n    var mapfn = aLen > 1 ? arguments[1] : undefined;\r\n    var mapping = mapfn !== undefined;\r\n    var index = 0;\r\n    var iterFn = getIterFn(O);\r\n    var length, result, step, iterator;\r\n    if (mapping) mapfn = ctx(mapfn, aLen > 2 ? arguments[2] : undefined, 2);\r\n    // if object isn't iterable or it's array with default iterator - use simple case\r\n    if (iterFn != undefined && !(C == Array && isArrayIter(iterFn))) {\r\n      for (iterator = iterFn.call(O), result = new C(); !(step = iterator.next()).done; index++) {\r\n        createProperty(result, index, mapping ? call(iterator, mapfn, [step.value, index], true) : step.value);\r\n      }\r\n    } else {\r\n      length = toLength(O.length);\r\n      for (result = new C(length); length > index; index++) {\r\n        createProperty(result, index, mapping ? mapfn(O[index], index) : O[index]);\r\n      }\r\n    }\r\n    result.length = index;\r\n    return result;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.from.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.index-of.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.index-of.js ***!
  \************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $indexOf = __webpack_require__(/*! ./_array-includes */ \"./node_modules/core-js/modules/_array-includes.js\")(false);\r\nvar $native = [].indexOf;\r\nvar NEGATIVE_ZERO = !!$native && 1 / [1].indexOf(1, -0) < 0;\r\n\r\n$export($export.P + $export.F * (NEGATIVE_ZERO || !__webpack_require__(/*! ./_strict-method */ \"./node_modules/core-js/modules/_strict-method.js\")($native)), 'Array', {\r\n  // 22.1.3.11 / 15.4.4.14 Array.prototype.indexOf(searchElement [, fromIndex])\r\n  indexOf: function indexOf(searchElement /* , fromIndex = 0 */) {\r\n    return NEGATIVE_ZERO\r\n      // convert -0 to +0\r\n      ? $native.apply(this, arguments) || 0\r\n      : $indexOf(this, searchElement, arguments[1]);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.index-of.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.is-array.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.is-array.js ***!
  \************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 22.1.2.2 / 15.4.3.2 Array.isArray(arg)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Array', { isArray: __webpack_require__(/*! ./_is-array */ \"./node_modules/core-js/modules/_is-array.js\") });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.is-array.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.iterator.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.iterator.js ***!
  \************************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar addToUnscopables = __webpack_require__(/*! ./_add-to-unscopables */ \"./node_modules/core-js/modules/_add-to-unscopables.js\");\r\nvar step = __webpack_require__(/*! ./_iter-step */ \"./node_modules/core-js/modules/_iter-step.js\");\r\nvar Iterators = __webpack_require__(/*! ./_iterators */ \"./node_modules/core-js/modules/_iterators.js\");\r\nvar toIObject = __webpack_require__(/*! ./_to-iobject */ \"./node_modules/core-js/modules/_to-iobject.js\");\r\n\r\n// 22.1.3.4 Array.prototype.entries()\r\n// 22.1.3.13 Array.prototype.keys()\r\n// 22.1.3.29 Array.prototype.values()\r\n// 22.1.3.30 Array.prototype[@@iterator]()\r\nmodule.exports = __webpack_require__(/*! ./_iter-define */ \"./node_modules/core-js/modules/_iter-define.js\")(Array, 'Array', function (iterated, kind) {\r\n  this._t = toIObject(iterated); // target\r\n  this._i = 0;                   // next index\r\n  this._k = kind;                // kind\r\n// 22.1.5.2.1 %ArrayIteratorPrototype%.next()\r\n}, function () {\r\n  var O = this._t;\r\n  var kind = this._k;\r\n  var index = this._i++;\r\n  if (!O || index >= O.length) {\r\n    this._t = undefined;\r\n    return step(1);\r\n  }\r\n  if (kind == 'keys') return step(0, index);\r\n  if (kind == 'values') return step(0, O[index]);\r\n  return step(0, [index, O[index]]);\r\n}, 'values');\r\n\r\n// argumentsList[@@iterator] is %ArrayProto_values% (9.4.4.6, 9.4.4.7)\r\nIterators.Arguments = Iterators.Array;\r\n\r\naddToUnscopables('keys');\r\naddToUnscopables('values');\r\naddToUnscopables('entries');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.iterator.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.join.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.join.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// 22.1.3.13 Array.prototype.join(separator)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar toIObject = __webpack_require__(/*! ./_to-iobject */ \"./node_modules/core-js/modules/_to-iobject.js\");\r\nvar arrayJoin = [].join;\r\n\r\n// fallback for not array-like strings\r\n$export($export.P + $export.F * (__webpack_require__(/*! ./_iobject */ \"./node_modules/core-js/modules/_iobject.js\") != Object || !__webpack_require__(/*! ./_strict-method */ \"./node_modules/core-js/modules/_strict-method.js\")(arrayJoin)), 'Array', {\r\n  join: function join(separator) {\r\n    return arrayJoin.call(toIObject(this), separator === undefined ? ',' : separator);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.join.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.last-index-of.js":
/*!*****************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.last-index-of.js ***!
  \*****************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar toIObject = __webpack_require__(/*! ./_to-iobject */ \"./node_modules/core-js/modules/_to-iobject.js\");\r\nvar toInteger = __webpack_require__(/*! ./_to-integer */ \"./node_modules/core-js/modules/_to-integer.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nvar $native = [].lastIndexOf;\r\nvar NEGATIVE_ZERO = !!$native && 1 / [1].lastIndexOf(1, -0) < 0;\r\n\r\n$export($export.P + $export.F * (NEGATIVE_ZERO || !__webpack_require__(/*! ./_strict-method */ \"./node_modules/core-js/modules/_strict-method.js\")($native)), 'Array', {\r\n  // 22.1.3.14 / 15.4.4.15 Array.prototype.lastIndexOf(searchElement [, fromIndex])\r\n  lastIndexOf: function lastIndexOf(searchElement /* , fromIndex = @[*-1] */) {\r\n    // convert -0 to +0\r\n    if (NEGATIVE_ZERO) return $native.apply(this, arguments) || 0;\r\n    var O = toIObject(this);\r\n    var length = toLength(O.length);\r\n    var index = length - 1;\r\n    if (arguments.length > 1) index = Math.min(index, toInteger(arguments[1]));\r\n    if (index < 0) index = length + index;\r\n    for (;index >= 0; index--) if (index in O) if (O[index] === searchElement) return index || 0;\r\n    return -1;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.last-index-of.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.map.js":
/*!*******************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.map.js ***!
  \*******************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $map = __webpack_require__(/*! ./_array-methods */ \"./node_modules/core-js/modules/_array-methods.js\")(1);\r\n\r\n$export($export.P + $export.F * !__webpack_require__(/*! ./_strict-method */ \"./node_modules/core-js/modules/_strict-method.js\")([].map, true), 'Array', {\r\n  // 22.1.3.15 / 15.4.4.19 Array.prototype.map(callbackfn [, thisArg])\r\n  map: function map(callbackfn /* , thisArg */) {\r\n    return $map(this, callbackfn, arguments[1]);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.map.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.of.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.of.js ***!
  \******************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar createProperty = __webpack_require__(/*! ./_create-property */ \"./node_modules/core-js/modules/_create-property.js\");\r\n\r\n// WebKit Array.of isn't generic\r\n$export($export.S + $export.F * __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\")(function () {\r\n  function F() { /* empty */ }\r\n  return !(Array.of.call(F) instanceof F);\r\n}), 'Array', {\r\n  // 22.1.2.3 Array.of( ...items)\r\n  of: function of(/* ...args */) {\r\n    var index = 0;\r\n    var aLen = arguments.length;\r\n    var result = new (typeof this == 'function' ? this : Array)(aLen);\r\n    while (aLen > index) createProperty(result, index, arguments[index++]);\r\n    result.length = aLen;\r\n    return result;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.of.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.reduce-right.js":
/*!****************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.reduce-right.js ***!
  \****************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $reduce = __webpack_require__(/*! ./_array-reduce */ \"./node_modules/core-js/modules/_array-reduce.js\");\r\n\r\n$export($export.P + $export.F * !__webpack_require__(/*! ./_strict-method */ \"./node_modules/core-js/modules/_strict-method.js\")([].reduceRight, true), 'Array', {\r\n  // 22.1.3.19 / 15.4.4.22 Array.prototype.reduceRight(callbackfn [, initialValue])\r\n  reduceRight: function reduceRight(callbackfn /* , initialValue */) {\r\n    return $reduce(this, callbackfn, arguments.length, arguments[1], true);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.reduce-right.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.reduce.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.reduce.js ***!
  \**********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $reduce = __webpack_require__(/*! ./_array-reduce */ \"./node_modules/core-js/modules/_array-reduce.js\");\r\n\r\n$export($export.P + $export.F * !__webpack_require__(/*! ./_strict-method */ \"./node_modules/core-js/modules/_strict-method.js\")([].reduce, true), 'Array', {\r\n  // 22.1.3.18 / 15.4.4.21 Array.prototype.reduce(callbackfn [, initialValue])\r\n  reduce: function reduce(callbackfn /* , initialValue */) {\r\n    return $reduce(this, callbackfn, arguments.length, arguments[1], false);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.reduce.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.slice.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.slice.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar html = __webpack_require__(/*! ./_html */ \"./node_modules/core-js/modules/_html.js\");\r\nvar cof = __webpack_require__(/*! ./_cof */ \"./node_modules/core-js/modules/_cof.js\");\r\nvar toAbsoluteIndex = __webpack_require__(/*! ./_to-absolute-index */ \"./node_modules/core-js/modules/_to-absolute-index.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nvar arraySlice = [].slice;\r\n\r\n// fallback for not array-like ES3 strings and DOM objects\r\n$export($export.P + $export.F * __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\")(function () {\r\n  if (html) arraySlice.call(html);\r\n}), 'Array', {\r\n  slice: function slice(begin, end) {\r\n    var len = toLength(this.length);\r\n    var klass = cof(this);\r\n    end = end === undefined ? len : end;\r\n    if (klass == 'Array') return arraySlice.call(this, begin, end);\r\n    var start = toAbsoluteIndex(begin, len);\r\n    var upTo = toAbsoluteIndex(end, len);\r\n    var size = toLength(upTo - start);\r\n    var cloned = new Array(size);\r\n    var i = 0;\r\n    for (; i < size; i++) cloned[i] = klass == 'String'\r\n      ? this.charAt(start + i)\r\n      : this[start + i];\r\n    return cloned;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.slice.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.some.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.some.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $some = __webpack_require__(/*! ./_array-methods */ \"./node_modules/core-js/modules/_array-methods.js\")(3);\r\n\r\n$export($export.P + $export.F * !__webpack_require__(/*! ./_strict-method */ \"./node_modules/core-js/modules/_strict-method.js\")([].some, true), 'Array', {\r\n  // 22.1.3.23 / 15.4.4.17 Array.prototype.some(callbackfn [, thisArg])\r\n  some: function some(callbackfn /* , thisArg */) {\r\n    return $some(this, callbackfn, arguments[1]);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.some.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.sort.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.sort.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar aFunction = __webpack_require__(/*! ./_a-function */ \"./node_modules/core-js/modules/_a-function.js\");\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar fails = __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\");\r\nvar $sort = [].sort;\r\nvar test = [1, 2, 3];\r\n\r\n$export($export.P + $export.F * (fails(function () {\r\n  // IE8-\r\n  test.sort(undefined);\r\n}) || !fails(function () {\r\n  // V8 bug\r\n  test.sort(null);\r\n  // Old WebKit\r\n}) || !__webpack_require__(/*! ./_strict-method */ \"./node_modules/core-js/modules/_strict-method.js\")($sort)), 'Array', {\r\n  // 22.1.3.25 Array.prototype.sort(comparefn)\r\n  sort: function sort(comparefn) {\r\n    return comparefn === undefined\r\n      ? $sort.call(toObject(this))\r\n      : $sort.call(toObject(this), aFunction(comparefn));\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.sort.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.array.species.js":
/*!***********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.array.species.js ***!
  \***********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("__webpack_require__(/*! ./_set-species */ \"./node_modules/core-js/modules/_set-species.js\")('Array');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.array.species.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.date.now.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/es6.date.now.js ***!
  \******************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.3.3.1 / 15.9.4.4 Date.now()\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Date', { now: function () { return new Date().getTime(); } });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.date.now.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.date.to-iso-string.js":
/*!****************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.date.to-iso-string.js ***!
  \****************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.3.4.36 / 15.9.5.43 Date.prototype.toISOString()\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar toISOString = __webpack_require__(/*! ./_date-to-iso-string */ \"./node_modules/core-js/modules/_date-to-iso-string.js\");\r\n\r\n// PhantomJS / old WebKit has a broken implementations\r\n$export($export.P + $export.F * (Date.prototype.toISOString !== toISOString), 'Date', {\r\n  toISOString: toISOString\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.date.to-iso-string.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.date.to-json.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.date.to-json.js ***!
  \**********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar toPrimitive = __webpack_require__(/*! ./_to-primitive */ \"./node_modules/core-js/modules/_to-primitive.js\");\r\n\r\n$export($export.P + $export.F * __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\")(function () {\r\n  return new Date(NaN).toJSON() !== null\r\n    || Date.prototype.toJSON.call({ toISOString: function () { return 1; } }) !== 1;\r\n}), 'Date', {\r\n  // eslint-disable-next-line no-unused-vars\r\n  toJSON: function toJSON(key) {\r\n    var O = toObject(this);\r\n    var pv = toPrimitive(O);\r\n    return typeof pv == 'number' && !isFinite(pv) ? null : O.toISOString();\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.date.to-json.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.date.to-primitive.js":
/*!***************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.date.to-primitive.js ***!
  \***************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var TO_PRIMITIVE = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('toPrimitive');\r\nvar proto = Date.prototype;\r\n\r\nif (!(TO_PRIMITIVE in proto)) __webpack_require__(/*! ./_hide */ \"./node_modules/core-js/modules/_hide.js\")(proto, TO_PRIMITIVE, __webpack_require__(/*! ./_date-to-primitive */ \"./node_modules/core-js/modules/_date-to-primitive.js\"));\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.date.to-primitive.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.date.to-string.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.date.to-string.js ***!
  \************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var DateProto = Date.prototype;\r\nvar INVALID_DATE = 'Invalid Date';\r\nvar TO_STRING = 'toString';\r\nvar $toString = DateProto[TO_STRING];\r\nvar getTime = DateProto.getTime;\r\nif (new Date(NaN) + '' != INVALID_DATE) {\r\n  __webpack_require__(/*! ./_redefine */ \"./node_modules/core-js/modules/_redefine.js\")(DateProto, TO_STRING, function toString() {\r\n    var value = getTime.call(this);\r\n    // eslint-disable-next-line no-self-compare\r\n    return value === value ? $toString.call(this) : INVALID_DATE;\r\n  });\r\n}\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.date.to-string.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.function.bind.js":
/*!***********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.function.bind.js ***!
  \***********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 19.2.3.2 / 15.3.4.5 Function.prototype.bind(thisArg, args...)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.P, 'Function', { bind: __webpack_require__(/*! ./_bind */ \"./node_modules/core-js/modules/_bind.js\") });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.function.bind.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.function.has-instance.js":
/*!*******************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.function.has-instance.js ***!
  \*******************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar getPrototypeOf = __webpack_require__(/*! ./_object-gpo */ \"./node_modules/core-js/modules/_object-gpo.js\");\r\nvar HAS_INSTANCE = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('hasInstance');\r\nvar FunctionProto = Function.prototype;\r\n// 19.2.3.6 Function.prototype[@@hasInstance](V)\r\nif (!(HAS_INSTANCE in FunctionProto)) __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\").f(FunctionProto, HAS_INSTANCE, { value: function (O) {\r\n  if (typeof this != 'function' || !isObject(O)) return false;\r\n  if (!isObject(this.prototype)) return O instanceof this;\r\n  // for environment w/o native `@@hasInstance` logic enough `instanceof`, but add this:\r\n  while (O = getPrototypeOf(O)) if (this.prototype === O) return true;\r\n  return false;\r\n} });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.function.has-instance.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.function.name.js":
/*!***********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.function.name.js ***!
  \***********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var dP = __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\").f;\r\nvar FProto = Function.prototype;\r\nvar nameRE = /^\\s*function ([^ (]*)/;\r\nvar NAME = 'name';\r\n\r\n// 19.2.4.2 name\r\nNAME in FProto || __webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\") && dP(FProto, NAME, {\r\n  configurable: true,\r\n  get: function () {\r\n    try {\r\n      return ('' + this).match(nameRE)[1];\r\n    } catch (e) {\r\n      return '';\r\n    }\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.function.name.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.map.js":
/*!*************************************************!*\
  !*** ./node_modules/core-js/modules/es6.map.js ***!
  \*************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar strong = __webpack_require__(/*! ./_collection-strong */ \"./node_modules/core-js/modules/_collection-strong.js\");\r\nvar validate = __webpack_require__(/*! ./_validate-collection */ \"./node_modules/core-js/modules/_validate-collection.js\");\r\nvar MAP = 'Map';\r\n\r\n// 23.1 Map Objects\r\nmodule.exports = __webpack_require__(/*! ./_collection */ \"./node_modules/core-js/modules/_collection.js\")(MAP, function (get) {\r\n  return function Map() { return get(this, arguments.length > 0 ? arguments[0] : undefined); };\r\n}, {\r\n  // 23.1.3.6 Map.prototype.get(key)\r\n  get: function get(key) {\r\n    var entry = strong.getEntry(validate(this, MAP), key);\r\n    return entry && entry.v;\r\n  },\r\n  // 23.1.3.9 Map.prototype.set(key, value)\r\n  set: function set(key, value) {\r\n    return strong.def(validate(this, MAP), key === 0 ? 0 : key, value);\r\n  }\r\n}, strong, true);\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.map.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.math.acosh.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.math.acosh.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.2.2.3 Math.acosh(x)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar log1p = __webpack_require__(/*! ./_math-log1p */ \"./node_modules/core-js/modules/_math-log1p.js\");\r\nvar sqrt = Math.sqrt;\r\nvar $acosh = Math.acosh;\r\n\r\n$export($export.S + $export.F * !($acosh\r\n  // V8 bug: https://code.google.com/p/v8/issues/detail?id=3509\r\n  && Math.floor($acosh(Number.MAX_VALUE)) == 710\r\n  // Tor Browser bug: Math.acosh(Infinity) -> NaN\r\n  && $acosh(Infinity) == Infinity\r\n), 'Math', {\r\n  acosh: function acosh(x) {\r\n    return (x = +x) < 1 ? NaN : x > 94906265.62425156\r\n      ? Math.log(x) + Math.LN2\r\n      : log1p(x - 1 + sqrt(x - 1) * sqrt(x + 1));\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.math.acosh.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.math.asinh.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.math.asinh.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.2.2.5 Math.asinh(x)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $asinh = Math.asinh;\r\n\r\nfunction asinh(x) {\r\n  return !isFinite(x = +x) || x == 0 ? x : x < 0 ? -asinh(-x) : Math.log(x + Math.sqrt(x * x + 1));\r\n}\r\n\r\n// Tor Browser bug: Math.asinh(0) -> -0\r\n$export($export.S + $export.F * !($asinh && 1 / $asinh(0) > 0), 'Math', { asinh: asinh });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.math.asinh.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.math.atanh.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.math.atanh.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.2.2.7 Math.atanh(x)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $atanh = Math.atanh;\r\n\r\n// Tor Browser bug: Math.atanh(-0) -> 0\r\n$export($export.S + $export.F * !($atanh && 1 / $atanh(-0) < 0), 'Math', {\r\n  atanh: function atanh(x) {\r\n    return (x = +x) == 0 ? x : Math.log((1 + x) / (1 - x)) / 2;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.math.atanh.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.math.cbrt.js":
/*!*******************************************************!*\
  !*** ./node_modules/core-js/modules/es6.math.cbrt.js ***!
  \*******************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.2.2.9 Math.cbrt(x)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar sign = __webpack_require__(/*! ./_math-sign */ \"./node_modules/core-js/modules/_math-sign.js\");\r\n\r\n$export($export.S, 'Math', {\r\n  cbrt: function cbrt(x) {\r\n    return sign(x = +x) * Math.pow(Math.abs(x), 1 / 3);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.math.cbrt.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.math.clz32.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.math.clz32.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.2.2.11 Math.clz32(x)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Math', {\r\n  clz32: function clz32(x) {\r\n    return (x >>>= 0) ? 31 - Math.floor(Math.log(x + 0.5) * Math.LOG2E) : 32;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.math.clz32.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.math.cosh.js":
/*!*******************************************************!*\
  !*** ./node_modules/core-js/modules/es6.math.cosh.js ***!
  \*******************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.2.2.12 Math.cosh(x)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar exp = Math.exp;\r\n\r\n$export($export.S, 'Math', {\r\n  cosh: function cosh(x) {\r\n    return (exp(x = +x) + exp(-x)) / 2;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.math.cosh.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.math.expm1.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.math.expm1.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.2.2.14 Math.expm1(x)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $expm1 = __webpack_require__(/*! ./_math-expm1 */ \"./node_modules/core-js/modules/_math-expm1.js\");\r\n\r\n$export($export.S + $export.F * ($expm1 != Math.expm1), 'Math', { expm1: $expm1 });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.math.expm1.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.math.fround.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.math.fround.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.2.2.16 Math.fround(x)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Math', { fround: __webpack_require__(/*! ./_math-fround */ \"./node_modules/core-js/modules/_math-fround.js\") });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.math.fround.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.math.hypot.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.math.hypot.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.2.2.17 Math.hypot([value1[, value2[, … ]]])\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar abs = Math.abs;\r\n\r\n$export($export.S, 'Math', {\r\n  hypot: function hypot(value1, value2) { // eslint-disable-line no-unused-vars\r\n    var sum = 0;\r\n    var i = 0;\r\n    var aLen = arguments.length;\r\n    var larg = 0;\r\n    var arg, div;\r\n    while (i < aLen) {\r\n      arg = abs(arguments[i++]);\r\n      if (larg < arg) {\r\n        div = larg / arg;\r\n        sum = sum * div * div + 1;\r\n        larg = arg;\r\n      } else if (arg > 0) {\r\n        div = arg / larg;\r\n        sum += div * div;\r\n      } else sum += arg;\r\n    }\r\n    return larg === Infinity ? Infinity : larg * Math.sqrt(sum);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.math.hypot.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.math.imul.js":
/*!*******************************************************!*\
  !*** ./node_modules/core-js/modules/es6.math.imul.js ***!
  \*******************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.2.2.18 Math.imul(x, y)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $imul = Math.imul;\r\n\r\n// some WebKit versions fails with big numbers, some has wrong arity\r\n$export($export.S + $export.F * __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\")(function () {\r\n  return $imul(0xffffffff, 5) != -5 || $imul.length != 2;\r\n}), 'Math', {\r\n  imul: function imul(x, y) {\r\n    var UINT16 = 0xffff;\r\n    var xn = +x;\r\n    var yn = +y;\r\n    var xl = UINT16 & xn;\r\n    var yl = UINT16 & yn;\r\n    return 0 | xl * yl + ((UINT16 & xn >>> 16) * yl + xl * (UINT16 & yn >>> 16) << 16 >>> 0);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.math.imul.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.math.log10.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.math.log10.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.2.2.21 Math.log10(x)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Math', {\r\n  log10: function log10(x) {\r\n    return Math.log(x) * Math.LOG10E;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.math.log10.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.math.log1p.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.math.log1p.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.2.2.20 Math.log1p(x)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Math', { log1p: __webpack_require__(/*! ./_math-log1p */ \"./node_modules/core-js/modules/_math-log1p.js\") });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.math.log1p.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.math.log2.js":
/*!*******************************************************!*\
  !*** ./node_modules/core-js/modules/es6.math.log2.js ***!
  \*******************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.2.2.22 Math.log2(x)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Math', {\r\n  log2: function log2(x) {\r\n    return Math.log(x) / Math.LN2;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.math.log2.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.math.sign.js":
/*!*******************************************************!*\
  !*** ./node_modules/core-js/modules/es6.math.sign.js ***!
  \*******************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.2.2.28 Math.sign(x)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Math', { sign: __webpack_require__(/*! ./_math-sign */ \"./node_modules/core-js/modules/_math-sign.js\") });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.math.sign.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.math.sinh.js":
/*!*******************************************************!*\
  !*** ./node_modules/core-js/modules/es6.math.sinh.js ***!
  \*******************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.2.2.30 Math.sinh(x)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar expm1 = __webpack_require__(/*! ./_math-expm1 */ \"./node_modules/core-js/modules/_math-expm1.js\");\r\nvar exp = Math.exp;\r\n\r\n// V8 near Chromium 38 has a problem with very small numbers\r\n$export($export.S + $export.F * __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\")(function () {\r\n  return !Math.sinh(-2e-17) != -2e-17;\r\n}), 'Math', {\r\n  sinh: function sinh(x) {\r\n    return Math.abs(x = +x) < 1\r\n      ? (expm1(x) - expm1(-x)) / 2\r\n      : (exp(x - 1) - exp(-x - 1)) * (Math.E / 2);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.math.sinh.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.math.tanh.js":
/*!*******************************************************!*\
  !*** ./node_modules/core-js/modules/es6.math.tanh.js ***!
  \*******************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.2.2.33 Math.tanh(x)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar expm1 = __webpack_require__(/*! ./_math-expm1 */ \"./node_modules/core-js/modules/_math-expm1.js\");\r\nvar exp = Math.exp;\r\n\r\n$export($export.S, 'Math', {\r\n  tanh: function tanh(x) {\r\n    var a = expm1(x = +x);\r\n    var b = expm1(-x);\r\n    return a == Infinity ? 1 : b == Infinity ? -1 : (a - b) / (exp(x) + exp(-x));\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.math.tanh.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.math.trunc.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.math.trunc.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.2.2.34 Math.trunc(x)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Math', {\r\n  trunc: function trunc(it) {\r\n    return (it > 0 ? Math.floor : Math.ceil)(it);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.math.trunc.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.number.constructor.js":
/*!****************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.number.constructor.js ***!
  \****************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar has = __webpack_require__(/*! ./_has */ \"./node_modules/core-js/modules/_has.js\");\r\nvar cof = __webpack_require__(/*! ./_cof */ \"./node_modules/core-js/modules/_cof.js\");\r\nvar inheritIfRequired = __webpack_require__(/*! ./_inherit-if-required */ \"./node_modules/core-js/modules/_inherit-if-required.js\");\r\nvar toPrimitive = __webpack_require__(/*! ./_to-primitive */ \"./node_modules/core-js/modules/_to-primitive.js\");\r\nvar fails = __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\");\r\nvar gOPN = __webpack_require__(/*! ./_object-gopn */ \"./node_modules/core-js/modules/_object-gopn.js\").f;\r\nvar gOPD = __webpack_require__(/*! ./_object-gopd */ \"./node_modules/core-js/modules/_object-gopd.js\").f;\r\nvar dP = __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\").f;\r\nvar $trim = __webpack_require__(/*! ./_string-trim */ \"./node_modules/core-js/modules/_string-trim.js\").trim;\r\nvar NUMBER = 'Number';\r\nvar $Number = global[NUMBER];\r\nvar Base = $Number;\r\nvar proto = $Number.prototype;\r\n// Opera ~12 has broken Object#toString\r\nvar BROKEN_COF = cof(__webpack_require__(/*! ./_object-create */ \"./node_modules/core-js/modules/_object-create.js\")(proto)) == NUMBER;\r\nvar TRIM = 'trim' in String.prototype;\r\n\r\n// 7.1.3 ToNumber(argument)\r\nvar toNumber = function (argument) {\r\n  var it = toPrimitive(argument, false);\r\n  if (typeof it == 'string' && it.length > 2) {\r\n    it = TRIM ? it.trim() : $trim(it, 3);\r\n    var first = it.charCodeAt(0);\r\n    var third, radix, maxCode;\r\n    if (first === 43 || first === 45) {\r\n      third = it.charCodeAt(2);\r\n      if (third === 88 || third === 120) return NaN; // Number('+0x1') should be NaN, old V8 fix\r\n    } else if (first === 48) {\r\n      switch (it.charCodeAt(1)) {\r\n        case 66: case 98: radix = 2; maxCode = 49; break; // fast equal /^0b[01]+$/i\r\n        case 79: case 111: radix = 8; maxCode = 55; break; // fast equal /^0o[0-7]+$/i\r\n        default: return +it;\r\n      }\r\n      for (var digits = it.slice(2), i = 0, l = digits.length, code; i < l; i++) {\r\n        code = digits.charCodeAt(i);\r\n        // parseInt parses a string to a first unavailable symbol\r\n        // but ToNumber should return NaN if a string contains unavailable symbols\r\n        if (code < 48 || code > maxCode) return NaN;\r\n      } return parseInt(digits, radix);\r\n    }\r\n  } return +it;\r\n};\r\n\r\nif (!$Number(' 0o1') || !$Number('0b1') || $Number('+0x1')) {\r\n  $Number = function Number(value) {\r\n    var it = arguments.length < 1 ? 0 : value;\r\n    var that = this;\r\n    return that instanceof $Number\r\n      // check on 1..constructor(foo) case\r\n      && (BROKEN_COF ? fails(function () { proto.valueOf.call(that); }) : cof(that) != NUMBER)\r\n        ? inheritIfRequired(new Base(toNumber(it)), that, $Number) : toNumber(it);\r\n  };\r\n  for (var keys = __webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\") ? gOPN(Base) : (\r\n    // ES3:\r\n    'MAX_VALUE,MIN_VALUE,NaN,NEGATIVE_INFINITY,POSITIVE_INFINITY,' +\r\n    // ES6 (in case, if modules with ES6 Number statics required before):\r\n    'EPSILON,isFinite,isInteger,isNaN,isSafeInteger,MAX_SAFE_INTEGER,' +\r\n    'MIN_SAFE_INTEGER,parseFloat,parseInt,isInteger'\r\n  ).split(','), j = 0, key; keys.length > j; j++) {\r\n    if (has(Base, key = keys[j]) && !has($Number, key)) {\r\n      dP($Number, key, gOPD(Base, key));\r\n    }\r\n  }\r\n  $Number.prototype = proto;\r\n  proto.constructor = $Number;\r\n  __webpack_require__(/*! ./_redefine */ \"./node_modules/core-js/modules/_redefine.js\")(global, NUMBER, $Number);\r\n}\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.number.constructor.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.number.epsilon.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.number.epsilon.js ***!
  \************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.1.2.1 Number.EPSILON\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Number', { EPSILON: Math.pow(2, -52) });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.number.epsilon.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.number.is-finite.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.number.is-finite.js ***!
  \**************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.1.2.2 Number.isFinite(number)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar _isFinite = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\").isFinite;\r\n\r\n$export($export.S, 'Number', {\r\n  isFinite: function isFinite(it) {\r\n    return typeof it == 'number' && _isFinite(it);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.number.is-finite.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.number.is-integer.js":
/*!***************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.number.is-integer.js ***!
  \***************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.1.2.3 Number.isInteger(number)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Number', { isInteger: __webpack_require__(/*! ./_is-integer */ \"./node_modules/core-js/modules/_is-integer.js\") });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.number.is-integer.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.number.is-nan.js":
/*!***********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.number.is-nan.js ***!
  \***********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.1.2.4 Number.isNaN(number)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Number', {\r\n  isNaN: function isNaN(number) {\r\n    // eslint-disable-next-line no-self-compare\r\n    return number != number;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.number.is-nan.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.number.is-safe-integer.js":
/*!********************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.number.is-safe-integer.js ***!
  \********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.1.2.5 Number.isSafeInteger(number)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar isInteger = __webpack_require__(/*! ./_is-integer */ \"./node_modules/core-js/modules/_is-integer.js\");\r\nvar abs = Math.abs;\r\n\r\n$export($export.S, 'Number', {\r\n  isSafeInteger: function isSafeInteger(number) {\r\n    return isInteger(number) && abs(number) <= 0x1fffffffffffff;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.number.is-safe-integer.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.number.max-safe-integer.js":
/*!*********************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.number.max-safe-integer.js ***!
  \*********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.1.2.6 Number.MAX_SAFE_INTEGER\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Number', { MAX_SAFE_INTEGER: 0x1fffffffffffff });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.number.max-safe-integer.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.number.min-safe-integer.js":
/*!*********************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.number.min-safe-integer.js ***!
  \*********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 20.1.2.10 Number.MIN_SAFE_INTEGER\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Number', { MIN_SAFE_INTEGER: -0x1fffffffffffff });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.number.min-safe-integer.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.number.parse-float.js":
/*!****************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.number.parse-float.js ***!
  \****************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $parseFloat = __webpack_require__(/*! ./_parse-float */ \"./node_modules/core-js/modules/_parse-float.js\");\r\n// 20.1.2.12 Number.parseFloat(string)\r\n$export($export.S + $export.F * (Number.parseFloat != $parseFloat), 'Number', { parseFloat: $parseFloat });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.number.parse-float.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.number.parse-int.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.number.parse-int.js ***!
  \**************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $parseInt = __webpack_require__(/*! ./_parse-int */ \"./node_modules/core-js/modules/_parse-int.js\");\r\n// 20.1.2.13 Number.parseInt(string, radix)\r\n$export($export.S + $export.F * (Number.parseInt != $parseInt), 'Number', { parseInt: $parseInt });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.number.parse-int.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.number.to-fixed.js":
/*!*************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.number.to-fixed.js ***!
  \*************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar toInteger = __webpack_require__(/*! ./_to-integer */ \"./node_modules/core-js/modules/_to-integer.js\");\r\nvar aNumberValue = __webpack_require__(/*! ./_a-number-value */ \"./node_modules/core-js/modules/_a-number-value.js\");\r\nvar repeat = __webpack_require__(/*! ./_string-repeat */ \"./node_modules/core-js/modules/_string-repeat.js\");\r\nvar $toFixed = 1.0.toFixed;\r\nvar floor = Math.floor;\r\nvar data = [0, 0, 0, 0, 0, 0];\r\nvar ERROR = 'Number.toFixed: incorrect invocation!';\r\nvar ZERO = '0';\r\n\r\nvar multiply = function (n, c) {\r\n  var i = -1;\r\n  var c2 = c;\r\n  while (++i < 6) {\r\n    c2 += n * data[i];\r\n    data[i] = c2 % 1e7;\r\n    c2 = floor(c2 / 1e7);\r\n  }\r\n};\r\nvar divide = function (n) {\r\n  var i = 6;\r\n  var c = 0;\r\n  while (--i >= 0) {\r\n    c += data[i];\r\n    data[i] = floor(c / n);\r\n    c = (c % n) * 1e7;\r\n  }\r\n};\r\nvar numToString = function () {\r\n  var i = 6;\r\n  var s = '';\r\n  while (--i >= 0) {\r\n    if (s !== '' || i === 0 || data[i] !== 0) {\r\n      var t = String(data[i]);\r\n      s = s === '' ? t : s + repeat.call(ZERO, 7 - t.length) + t;\r\n    }\r\n  } return s;\r\n};\r\nvar pow = function (x, n, acc) {\r\n  return n === 0 ? acc : n % 2 === 1 ? pow(x, n - 1, acc * x) : pow(x * x, n / 2, acc);\r\n};\r\nvar log = function (x) {\r\n  var n = 0;\r\n  var x2 = x;\r\n  while (x2 >= 4096) {\r\n    n += 12;\r\n    x2 /= 4096;\r\n  }\r\n  while (x2 >= 2) {\r\n    n += 1;\r\n    x2 /= 2;\r\n  } return n;\r\n};\r\n\r\n$export($export.P + $export.F * (!!$toFixed && (\r\n  0.00008.toFixed(3) !== '0.000' ||\r\n  0.9.toFixed(0) !== '1' ||\r\n  1.255.toFixed(2) !== '1.25' ||\r\n  1000000000000000128.0.toFixed(0) !== '1000000000000000128'\r\n) || !__webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\")(function () {\r\n  // V8 ~ Android 4.3-\r\n  $toFixed.call({});\r\n})), 'Number', {\r\n  toFixed: function toFixed(fractionDigits) {\r\n    var x = aNumberValue(this, ERROR);\r\n    var f = toInteger(fractionDigits);\r\n    var s = '';\r\n    var m = ZERO;\r\n    var e, z, j, k;\r\n    if (f < 0 || f > 20) throw RangeError(ERROR);\r\n    // eslint-disable-next-line no-self-compare\r\n    if (x != x) return 'NaN';\r\n    if (x <= -1e21 || x >= 1e21) return String(x);\r\n    if (x < 0) {\r\n      s = '-';\r\n      x = -x;\r\n    }\r\n    if (x > 1e-21) {\r\n      e = log(x * pow(2, 69, 1)) - 69;\r\n      z = e < 0 ? x * pow(2, -e, 1) : x / pow(2, e, 1);\r\n      z *= 0x10000000000000;\r\n      e = 52 - e;\r\n      if (e > 0) {\r\n        multiply(0, z);\r\n        j = f;\r\n        while (j >= 7) {\r\n          multiply(1e7, 0);\r\n          j -= 7;\r\n        }\r\n        multiply(pow(10, j, 1), 0);\r\n        j = e - 1;\r\n        while (j >= 23) {\r\n          divide(1 << 23);\r\n          j -= 23;\r\n        }\r\n        divide(1 << j);\r\n        multiply(1, 1);\r\n        divide(2);\r\n        m = numToString();\r\n      } else {\r\n        multiply(0, z);\r\n        multiply(1 << -e, 0);\r\n        m = numToString() + repeat.call(ZERO, f);\r\n      }\r\n    }\r\n    if (f > 0) {\r\n      k = m.length;\r\n      m = s + (k <= f ? '0.' + repeat.call(ZERO, f - k) + m : m.slice(0, k - f) + '.' + m.slice(k - f));\r\n    } else {\r\n      m = s + m;\r\n    } return m;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.number.to-fixed.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.number.to-precision.js":
/*!*****************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.number.to-precision.js ***!
  \*****************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $fails = __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\");\r\nvar aNumberValue = __webpack_require__(/*! ./_a-number-value */ \"./node_modules/core-js/modules/_a-number-value.js\");\r\nvar $toPrecision = 1.0.toPrecision;\r\n\r\n$export($export.P + $export.F * ($fails(function () {\r\n  // IE7-\r\n  return $toPrecision.call(1, undefined) !== '1';\r\n}) || !$fails(function () {\r\n  // V8 ~ Android 4.3-\r\n  $toPrecision.call({});\r\n})), 'Number', {\r\n  toPrecision: function toPrecision(precision) {\r\n    var that = aNumberValue(this, 'Number#toPrecision: incorrect invocation!');\r\n    return precision === undefined ? $toPrecision.call(that) : $toPrecision.call(that, precision);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.number.to-precision.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.object.assign.js":
/*!***********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.object.assign.js ***!
  \***********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 19.1.3.1 Object.assign(target, source)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S + $export.F, 'Object', { assign: __webpack_require__(/*! ./_object-assign */ \"./node_modules/core-js/modules/_object-assign.js\") });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.object.assign.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.object.create.js":
/*!***********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.object.create.js ***!
  \***********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n// 19.1.2.2 / 15.2.3.5 Object.create(O [, Properties])\r\n$export($export.S, 'Object', { create: __webpack_require__(/*! ./_object-create */ \"./node_modules/core-js/modules/_object-create.js\") });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.object.create.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.object.define-properties.js":
/*!**********************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.object.define-properties.js ***!
  \**********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n// 19.1.2.3 / 15.2.3.7 Object.defineProperties(O, Properties)\r\n$export($export.S + $export.F * !__webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\"), 'Object', { defineProperties: __webpack_require__(/*! ./_object-dps */ \"./node_modules/core-js/modules/_object-dps.js\") });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.object.define-properties.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.object.define-property.js":
/*!********************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.object.define-property.js ***!
  \********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n// 19.1.2.4 / 15.2.3.6 Object.defineProperty(O, P, Attributes)\r\n$export($export.S + $export.F * !__webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\"), 'Object', { defineProperty: __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\").f });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.object.define-property.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.object.freeze.js":
/*!***********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.object.freeze.js ***!
  \***********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 19.1.2.5 Object.freeze(O)\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar meta = __webpack_require__(/*! ./_meta */ \"./node_modules/core-js/modules/_meta.js\").onFreeze;\r\n\r\n__webpack_require__(/*! ./_object-sap */ \"./node_modules/core-js/modules/_object-sap.js\")('freeze', function ($freeze) {\r\n  return function freeze(it) {\r\n    return $freeze && isObject(it) ? $freeze(meta(it)) : it;\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.object.freeze.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.object.get-own-property-descriptor.js":
/*!********************************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.object.get-own-property-descriptor.js ***!
  \********************************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 19.1.2.6 Object.getOwnPropertyDescriptor(O, P)\r\nvar toIObject = __webpack_require__(/*! ./_to-iobject */ \"./node_modules/core-js/modules/_to-iobject.js\");\r\nvar $getOwnPropertyDescriptor = __webpack_require__(/*! ./_object-gopd */ \"./node_modules/core-js/modules/_object-gopd.js\").f;\r\n\r\n__webpack_require__(/*! ./_object-sap */ \"./node_modules/core-js/modules/_object-sap.js\")('getOwnPropertyDescriptor', function () {\r\n  return function getOwnPropertyDescriptor(it, key) {\r\n    return $getOwnPropertyDescriptor(toIObject(it), key);\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.object.get-own-property-descriptor.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.object.get-own-property-names.js":
/*!***************************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.object.get-own-property-names.js ***!
  \***************************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 19.1.2.7 Object.getOwnPropertyNames(O)\r\n__webpack_require__(/*! ./_object-sap */ \"./node_modules/core-js/modules/_object-sap.js\")('getOwnPropertyNames', function () {\r\n  return __webpack_require__(/*! ./_object-gopn-ext */ \"./node_modules/core-js/modules/_object-gopn-ext.js\").f;\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.object.get-own-property-names.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.object.get-prototype-of.js":
/*!*********************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.object.get-prototype-of.js ***!
  \*********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 19.1.2.9 Object.getPrototypeOf(O)\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar $getPrototypeOf = __webpack_require__(/*! ./_object-gpo */ \"./node_modules/core-js/modules/_object-gpo.js\");\r\n\r\n__webpack_require__(/*! ./_object-sap */ \"./node_modules/core-js/modules/_object-sap.js\")('getPrototypeOf', function () {\r\n  return function getPrototypeOf(it) {\r\n    return $getPrototypeOf(toObject(it));\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.object.get-prototype-of.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.object.is-extensible.js":
/*!******************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.object.is-extensible.js ***!
  \******************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 19.1.2.11 Object.isExtensible(O)\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\n\r\n__webpack_require__(/*! ./_object-sap */ \"./node_modules/core-js/modules/_object-sap.js\")('isExtensible', function ($isExtensible) {\r\n  return function isExtensible(it) {\r\n    return isObject(it) ? $isExtensible ? $isExtensible(it) : true : false;\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.object.is-extensible.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.object.is-frozen.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.object.is-frozen.js ***!
  \**************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 19.1.2.12 Object.isFrozen(O)\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\n\r\n__webpack_require__(/*! ./_object-sap */ \"./node_modules/core-js/modules/_object-sap.js\")('isFrozen', function ($isFrozen) {\r\n  return function isFrozen(it) {\r\n    return isObject(it) ? $isFrozen ? $isFrozen(it) : false : true;\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.object.is-frozen.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.object.is-sealed.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.object.is-sealed.js ***!
  \**************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 19.1.2.13 Object.isSealed(O)\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\n\r\n__webpack_require__(/*! ./_object-sap */ \"./node_modules/core-js/modules/_object-sap.js\")('isSealed', function ($isSealed) {\r\n  return function isSealed(it) {\r\n    return isObject(it) ? $isSealed ? $isSealed(it) : false : true;\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.object.is-sealed.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.object.is.js":
/*!*******************************************************!*\
  !*** ./node_modules/core-js/modules/es6.object.is.js ***!
  \*******************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 19.1.3.10 Object.is(value1, value2)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n$export($export.S, 'Object', { is: __webpack_require__(/*! ./_same-value */ \"./node_modules/core-js/modules/_same-value.js\") });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.object.is.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.object.keys.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.object.keys.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 19.1.2.14 Object.keys(O)\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar $keys = __webpack_require__(/*! ./_object-keys */ \"./node_modules/core-js/modules/_object-keys.js\");\r\n\r\n__webpack_require__(/*! ./_object-sap */ \"./node_modules/core-js/modules/_object-sap.js\")('keys', function () {\r\n  return function keys(it) {\r\n    return $keys(toObject(it));\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.object.keys.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.object.prevent-extensions.js":
/*!***********************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.object.prevent-extensions.js ***!
  \***********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 19.1.2.15 Object.preventExtensions(O)\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar meta = __webpack_require__(/*! ./_meta */ \"./node_modules/core-js/modules/_meta.js\").onFreeze;\r\n\r\n__webpack_require__(/*! ./_object-sap */ \"./node_modules/core-js/modules/_object-sap.js\")('preventExtensions', function ($preventExtensions) {\r\n  return function preventExtensions(it) {\r\n    return $preventExtensions && isObject(it) ? $preventExtensions(meta(it)) : it;\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.object.prevent-extensions.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.object.seal.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.object.seal.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 19.1.2.17 Object.seal(O)\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar meta = __webpack_require__(/*! ./_meta */ \"./node_modules/core-js/modules/_meta.js\").onFreeze;\r\n\r\n__webpack_require__(/*! ./_object-sap */ \"./node_modules/core-js/modules/_object-sap.js\")('seal', function ($seal) {\r\n  return function seal(it) {\r\n    return $seal && isObject(it) ? $seal(meta(it)) : it;\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.object.seal.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.object.set-prototype-of.js":
/*!*********************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.object.set-prototype-of.js ***!
  \*********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 19.1.3.19 Object.setPrototypeOf(O, proto)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n$export($export.S, 'Object', { setPrototypeOf: __webpack_require__(/*! ./_set-proto */ \"./node_modules/core-js/modules/_set-proto.js\").set });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.object.set-prototype-of.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.object.to-string.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.object.to-string.js ***!
  \**************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// 19.1.3.6 Object.prototype.toString()\r\nvar classof = __webpack_require__(/*! ./_classof */ \"./node_modules/core-js/modules/_classof.js\");\r\nvar test = {};\r\ntest[__webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('toStringTag')] = 'z';\r\nif (test + '' != '[object z]') {\r\n  __webpack_require__(/*! ./_redefine */ \"./node_modules/core-js/modules/_redefine.js\")(Object.prototype, 'toString', function toString() {\r\n    return '[object ' + classof(this) + ']';\r\n  }, true);\r\n}\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.object.to-string.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.parse-float.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.parse-float.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $parseFloat = __webpack_require__(/*! ./_parse-float */ \"./node_modules/core-js/modules/_parse-float.js\");\r\n// 18.2.4 parseFloat(string)\r\n$export($export.G + $export.F * (parseFloat != $parseFloat), { parseFloat: $parseFloat });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.parse-float.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.parse-int.js":
/*!*******************************************************!*\
  !*** ./node_modules/core-js/modules/es6.parse-int.js ***!
  \*******************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $parseInt = __webpack_require__(/*! ./_parse-int */ \"./node_modules/core-js/modules/_parse-int.js\");\r\n// 18.2.5 parseInt(string, radix)\r\n$export($export.G + $export.F * (parseInt != $parseInt), { parseInt: $parseInt });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.parse-int.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.promise.js":
/*!*****************************************************!*\
  !*** ./node_modules/core-js/modules/es6.promise.js ***!
  \*****************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar LIBRARY = __webpack_require__(/*! ./_library */ \"./node_modules/core-js/modules/_library.js\");\r\nvar global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar ctx = __webpack_require__(/*! ./_ctx */ \"./node_modules/core-js/modules/_ctx.js\");\r\nvar classof = __webpack_require__(/*! ./_classof */ \"./node_modules/core-js/modules/_classof.js\");\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar aFunction = __webpack_require__(/*! ./_a-function */ \"./node_modules/core-js/modules/_a-function.js\");\r\nvar anInstance = __webpack_require__(/*! ./_an-instance */ \"./node_modules/core-js/modules/_an-instance.js\");\r\nvar forOf = __webpack_require__(/*! ./_for-of */ \"./node_modules/core-js/modules/_for-of.js\");\r\nvar speciesConstructor = __webpack_require__(/*! ./_species-constructor */ \"./node_modules/core-js/modules/_species-constructor.js\");\r\nvar task = __webpack_require__(/*! ./_task */ \"./node_modules/core-js/modules/_task.js\").set;\r\nvar microtask = __webpack_require__(/*! ./_microtask */ \"./node_modules/core-js/modules/_microtask.js\")();\r\nvar newPromiseCapabilityModule = __webpack_require__(/*! ./_new-promise-capability */ \"./node_modules/core-js/modules/_new-promise-capability.js\");\r\nvar perform = __webpack_require__(/*! ./_perform */ \"./node_modules/core-js/modules/_perform.js\");\r\nvar userAgent = __webpack_require__(/*! ./_user-agent */ \"./node_modules/core-js/modules/_user-agent.js\");\r\nvar promiseResolve = __webpack_require__(/*! ./_promise-resolve */ \"./node_modules/core-js/modules/_promise-resolve.js\");\r\nvar PROMISE = 'Promise';\r\nvar TypeError = global.TypeError;\r\nvar process = global.process;\r\nvar versions = process && process.versions;\r\nvar v8 = versions && versions.v8 || '';\r\nvar $Promise = global[PROMISE];\r\nvar isNode = classof(process) == 'process';\r\nvar empty = function () { /* empty */ };\r\nvar Internal, newGenericPromiseCapability, OwnPromiseCapability, Wrapper;\r\nvar newPromiseCapability = newGenericPromiseCapability = newPromiseCapabilityModule.f;\r\n\r\nvar USE_NATIVE = !!function () {\r\n  try {\r\n    // correct subclassing with @@species support\r\n    var promise = $Promise.resolve(1);\r\n    var FakePromise = (promise.constructor = {})[__webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('species')] = function (exec) {\r\n      exec(empty, empty);\r\n    };\r\n    // unhandled rejections tracking support, NodeJS Promise without it fails @@species test\r\n    return (isNode || typeof PromiseRejectionEvent == 'function')\r\n      && promise.then(empty) instanceof FakePromise\r\n      // v8 6.6 (Node 10 and Chrome 66) have a bug with resolving custom thenables\r\n      // https://bugs.chromium.org/p/chromium/issues/detail?id=830565\r\n      // we can't detect it synchronously, so just check versions\r\n      && v8.indexOf('6.6') !== 0\r\n      && userAgent.indexOf('Chrome/66') === -1;\r\n  } catch (e) { /* empty */ }\r\n}();\r\n\r\n// helpers\r\nvar isThenable = function (it) {\r\n  var then;\r\n  return isObject(it) && typeof (then = it.then) == 'function' ? then : false;\r\n};\r\nvar notify = function (promise, isReject) {\r\n  if (promise._n) return;\r\n  promise._n = true;\r\n  var chain = promise._c;\r\n  microtask(function () {\r\n    var value = promise._v;\r\n    var ok = promise._s == 1;\r\n    var i = 0;\r\n    var run = function (reaction) {\r\n      var handler = ok ? reaction.ok : reaction.fail;\r\n      var resolve = reaction.resolve;\r\n      var reject = reaction.reject;\r\n      var domain = reaction.domain;\r\n      var result, then, exited;\r\n      try {\r\n        if (handler) {\r\n          if (!ok) {\r\n            if (promise._h == 2) onHandleUnhandled(promise);\r\n            promise._h = 1;\r\n          }\r\n          if (handler === true) result = value;\r\n          else {\r\n            if (domain) domain.enter();\r\n            result = handler(value); // may throw\r\n            if (domain) {\r\n              domain.exit();\r\n              exited = true;\r\n            }\r\n          }\r\n          if (result === reaction.promise) {\r\n            reject(TypeError('Promise-chain cycle'));\r\n          } else if (then = isThenable(result)) {\r\n            then.call(result, resolve, reject);\r\n          } else resolve(result);\r\n        } else reject(value);\r\n      } catch (e) {\r\n        if (domain && !exited) domain.exit();\r\n        reject(e);\r\n      }\r\n    };\r\n    while (chain.length > i) run(chain[i++]); // variable length - can't use forEach\r\n    promise._c = [];\r\n    promise._n = false;\r\n    if (isReject && !promise._h) onUnhandled(promise);\r\n  });\r\n};\r\nvar onUnhandled = function (promise) {\r\n  task.call(global, function () {\r\n    var value = promise._v;\r\n    var unhandled = isUnhandled(promise);\r\n    var result, handler, console;\r\n    if (unhandled) {\r\n      result = perform(function () {\r\n        if (isNode) {\r\n          process.emit('unhandledRejection', value, promise);\r\n        } else if (handler = global.onunhandledrejection) {\r\n          handler({ promise: promise, reason: value });\r\n        } else if ((console = global.console) && console.error) {\r\n          console.error('Unhandled promise rejection', value);\r\n        }\r\n      });\r\n      // Browsers should not trigger `rejectionHandled` event if it was handled here, NodeJS - should\r\n      promise._h = isNode || isUnhandled(promise) ? 2 : 1;\r\n    } promise._a = undefined;\r\n    if (unhandled && result.e) throw result.v;\r\n  });\r\n};\r\nvar isUnhandled = function (promise) {\r\n  return promise._h !== 1 && (promise._a || promise._c).length === 0;\r\n};\r\nvar onHandleUnhandled = function (promise) {\r\n  task.call(global, function () {\r\n    var handler;\r\n    if (isNode) {\r\n      process.emit('rejectionHandled', promise);\r\n    } else if (handler = global.onrejectionhandled) {\r\n      handler({ promise: promise, reason: promise._v });\r\n    }\r\n  });\r\n};\r\nvar $reject = function (value) {\r\n  var promise = this;\r\n  if (promise._d) return;\r\n  promise._d = true;\r\n  promise = promise._w || promise; // unwrap\r\n  promise._v = value;\r\n  promise._s = 2;\r\n  if (!promise._a) promise._a = promise._c.slice();\r\n  notify(promise, true);\r\n};\r\nvar $resolve = function (value) {\r\n  var promise = this;\r\n  var then;\r\n  if (promise._d) return;\r\n  promise._d = true;\r\n  promise = promise._w || promise; // unwrap\r\n  try {\r\n    if (promise === value) throw TypeError(\"Promise can't be resolved itself\");\r\n    if (then = isThenable(value)) {\r\n      microtask(function () {\r\n        var wrapper = { _w: promise, _d: false }; // wrap\r\n        try {\r\n          then.call(value, ctx($resolve, wrapper, 1), ctx($reject, wrapper, 1));\r\n        } catch (e) {\r\n          $reject.call(wrapper, e);\r\n        }\r\n      });\r\n    } else {\r\n      promise._v = value;\r\n      promise._s = 1;\r\n      notify(promise, false);\r\n    }\r\n  } catch (e) {\r\n    $reject.call({ _w: promise, _d: false }, e); // wrap\r\n  }\r\n};\r\n\r\n// constructor polyfill\r\nif (!USE_NATIVE) {\r\n  // 25.4.3.1 Promise(executor)\r\n  $Promise = function Promise(executor) {\r\n    anInstance(this, $Promise, PROMISE, '_h');\r\n    aFunction(executor);\r\n    Internal.call(this);\r\n    try {\r\n      executor(ctx($resolve, this, 1), ctx($reject, this, 1));\r\n    } catch (err) {\r\n      $reject.call(this, err);\r\n    }\r\n  };\r\n  // eslint-disable-next-line no-unused-vars\r\n  Internal = function Promise(executor) {\r\n    this._c = [];             // <- awaiting reactions\r\n    this._a = undefined;      // <- checked in isUnhandled reactions\r\n    this._s = 0;              // <- state\r\n    this._d = false;          // <- done\r\n    this._v = undefined;      // <- value\r\n    this._h = 0;              // <- rejection state, 0 - default, 1 - handled, 2 - unhandled\r\n    this._n = false;          // <- notify\r\n  };\r\n  Internal.prototype = __webpack_require__(/*! ./_redefine-all */ \"./node_modules/core-js/modules/_redefine-all.js\")($Promise.prototype, {\r\n    // 25.4.5.3 Promise.prototype.then(onFulfilled, onRejected)\r\n    then: function then(onFulfilled, onRejected) {\r\n      var reaction = newPromiseCapability(speciesConstructor(this, $Promise));\r\n      reaction.ok = typeof onFulfilled == 'function' ? onFulfilled : true;\r\n      reaction.fail = typeof onRejected == 'function' && onRejected;\r\n      reaction.domain = isNode ? process.domain : undefined;\r\n      this._c.push(reaction);\r\n      if (this._a) this._a.push(reaction);\r\n      if (this._s) notify(this, false);\r\n      return reaction.promise;\r\n    },\r\n    // 25.4.5.1 Promise.prototype.catch(onRejected)\r\n    'catch': function (onRejected) {\r\n      return this.then(undefined, onRejected);\r\n    }\r\n  });\r\n  OwnPromiseCapability = function () {\r\n    var promise = new Internal();\r\n    this.promise = promise;\r\n    this.resolve = ctx($resolve, promise, 1);\r\n    this.reject = ctx($reject, promise, 1);\r\n  };\r\n  newPromiseCapabilityModule.f = newPromiseCapability = function (C) {\r\n    return C === $Promise || C === Wrapper\r\n      ? new OwnPromiseCapability(C)\r\n      : newGenericPromiseCapability(C);\r\n  };\r\n}\r\n\r\n$export($export.G + $export.W + $export.F * !USE_NATIVE, { Promise: $Promise });\r\n__webpack_require__(/*! ./_set-to-string-tag */ \"./node_modules/core-js/modules/_set-to-string-tag.js\")($Promise, PROMISE);\r\n__webpack_require__(/*! ./_set-species */ \"./node_modules/core-js/modules/_set-species.js\")(PROMISE);\r\nWrapper = __webpack_require__(/*! ./_core */ \"./node_modules/core-js/modules/_core.js\")[PROMISE];\r\n\r\n// statics\r\n$export($export.S + $export.F * !USE_NATIVE, PROMISE, {\r\n  // 25.4.4.5 Promise.reject(r)\r\n  reject: function reject(r) {\r\n    var capability = newPromiseCapability(this);\r\n    var $$reject = capability.reject;\r\n    $$reject(r);\r\n    return capability.promise;\r\n  }\r\n});\r\n$export($export.S + $export.F * (LIBRARY || !USE_NATIVE), PROMISE, {\r\n  // 25.4.4.6 Promise.resolve(x)\r\n  resolve: function resolve(x) {\r\n    return promiseResolve(LIBRARY && this === Wrapper ? $Promise : this, x);\r\n  }\r\n});\r\n$export($export.S + $export.F * !(USE_NATIVE && __webpack_require__(/*! ./_iter-detect */ \"./node_modules/core-js/modules/_iter-detect.js\")(function (iter) {\r\n  $Promise.all(iter)['catch'](empty);\r\n})), PROMISE, {\r\n  // 25.4.4.1 Promise.all(iterable)\r\n  all: function all(iterable) {\r\n    var C = this;\r\n    var capability = newPromiseCapability(C);\r\n    var resolve = capability.resolve;\r\n    var reject = capability.reject;\r\n    var result = perform(function () {\r\n      var values = [];\r\n      var index = 0;\r\n      var remaining = 1;\r\n      forOf(iterable, false, function (promise) {\r\n        var $index = index++;\r\n        var alreadyCalled = false;\r\n        values.push(undefined);\r\n        remaining++;\r\n        C.resolve(promise).then(function (value) {\r\n          if (alreadyCalled) return;\r\n          alreadyCalled = true;\r\n          values[$index] = value;\r\n          --remaining || resolve(values);\r\n        }, reject);\r\n      });\r\n      --remaining || resolve(values);\r\n    });\r\n    if (result.e) reject(result.v);\r\n    return capability.promise;\r\n  },\r\n  // 25.4.4.4 Promise.race(iterable)\r\n  race: function race(iterable) {\r\n    var C = this;\r\n    var capability = newPromiseCapability(C);\r\n    var reject = capability.reject;\r\n    var result = perform(function () {\r\n      forOf(iterable, false, function (promise) {\r\n        C.resolve(promise).then(capability.resolve, reject);\r\n      });\r\n    });\r\n    if (result.e) reject(result.v);\r\n    return capability.promise;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.promise.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.reflect.apply.js":
/*!***********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.reflect.apply.js ***!
  \***********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 26.1.1 Reflect.apply(target, thisArgument, argumentsList)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar aFunction = __webpack_require__(/*! ./_a-function */ \"./node_modules/core-js/modules/_a-function.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar rApply = (__webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\").Reflect || {}).apply;\r\nvar fApply = Function.apply;\r\n// MS Edge argumentsList argument is optional\r\n$export($export.S + $export.F * !__webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\")(function () {\r\n  rApply(function () { /* empty */ });\r\n}), 'Reflect', {\r\n  apply: function apply(target, thisArgument, argumentsList) {\r\n    var T = aFunction(target);\r\n    var L = anObject(argumentsList);\r\n    return rApply ? rApply(T, thisArgument, L) : fApply.call(T, thisArgument, L);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.reflect.apply.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.reflect.construct.js":
/*!***************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.reflect.construct.js ***!
  \***************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 26.1.2 Reflect.construct(target, argumentsList [, newTarget])\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar create = __webpack_require__(/*! ./_object-create */ \"./node_modules/core-js/modules/_object-create.js\");\r\nvar aFunction = __webpack_require__(/*! ./_a-function */ \"./node_modules/core-js/modules/_a-function.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar fails = __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\");\r\nvar bind = __webpack_require__(/*! ./_bind */ \"./node_modules/core-js/modules/_bind.js\");\r\nvar rConstruct = (__webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\").Reflect || {}).construct;\r\n\r\n// MS Edge supports only 2 arguments and argumentsList argument is optional\r\n// FF Nightly sets third argument as `new.target`, but does not create `this` from it\r\nvar NEW_TARGET_BUG = fails(function () {\r\n  function F() { /* empty */ }\r\n  return !(rConstruct(function () { /* empty */ }, [], F) instanceof F);\r\n});\r\nvar ARGS_BUG = !fails(function () {\r\n  rConstruct(function () { /* empty */ });\r\n});\r\n\r\n$export($export.S + $export.F * (NEW_TARGET_BUG || ARGS_BUG), 'Reflect', {\r\n  construct: function construct(Target, args /* , newTarget */) {\r\n    aFunction(Target);\r\n    anObject(args);\r\n    var newTarget = arguments.length < 3 ? Target : aFunction(arguments[2]);\r\n    if (ARGS_BUG && !NEW_TARGET_BUG) return rConstruct(Target, args, newTarget);\r\n    if (Target == newTarget) {\r\n      // w/o altered newTarget, optimization for 0-4 arguments\r\n      switch (args.length) {\r\n        case 0: return new Target();\r\n        case 1: return new Target(args[0]);\r\n        case 2: return new Target(args[0], args[1]);\r\n        case 3: return new Target(args[0], args[1], args[2]);\r\n        case 4: return new Target(args[0], args[1], args[2], args[3]);\r\n      }\r\n      // w/o altered newTarget, lot of arguments case\r\n      var $args = [null];\r\n      $args.push.apply($args, args);\r\n      return new (bind.apply(Target, $args))();\r\n    }\r\n    // with altered newTarget, not support built-in constructors\r\n    var proto = newTarget.prototype;\r\n    var instance = create(isObject(proto) ? proto : Object.prototype);\r\n    var result = Function.apply.call(Target, instance, args);\r\n    return isObject(result) ? result : instance;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.reflect.construct.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.reflect.define-property.js":
/*!*********************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.reflect.define-property.js ***!
  \*********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 26.1.3 Reflect.defineProperty(target, propertyKey, attributes)\r\nvar dP = __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\");\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar toPrimitive = __webpack_require__(/*! ./_to-primitive */ \"./node_modules/core-js/modules/_to-primitive.js\");\r\n\r\n// MS Edge has broken Reflect.defineProperty - throwing instead of returning false\r\n$export($export.S + $export.F * __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\")(function () {\r\n  // eslint-disable-next-line no-undef\r\n  Reflect.defineProperty(dP.f({}, 1, { value: 1 }), 1, { value: 2 });\r\n}), 'Reflect', {\r\n  defineProperty: function defineProperty(target, propertyKey, attributes) {\r\n    anObject(target);\r\n    propertyKey = toPrimitive(propertyKey, true);\r\n    anObject(attributes);\r\n    try {\r\n      dP.f(target, propertyKey, attributes);\r\n      return true;\r\n    } catch (e) {\r\n      return false;\r\n    }\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.reflect.define-property.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.reflect.delete-property.js":
/*!*********************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.reflect.delete-property.js ***!
  \*********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 26.1.4 Reflect.deleteProperty(target, propertyKey)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar gOPD = __webpack_require__(/*! ./_object-gopd */ \"./node_modules/core-js/modules/_object-gopd.js\").f;\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\n\r\n$export($export.S, 'Reflect', {\r\n  deleteProperty: function deleteProperty(target, propertyKey) {\r\n    var desc = gOPD(anObject(target), propertyKey);\r\n    return desc && !desc.configurable ? false : delete target[propertyKey];\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.reflect.delete-property.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.reflect.enumerate.js":
/*!***************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.reflect.enumerate.js ***!
  \***************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// 26.1.5 Reflect.enumerate(target)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar Enumerate = function (iterated) {\r\n  this._t = anObject(iterated); // target\r\n  this._i = 0;                  // next index\r\n  var keys = this._k = [];      // keys\r\n  var key;\r\n  for (key in iterated) keys.push(key);\r\n};\r\n__webpack_require__(/*! ./_iter-create */ \"./node_modules/core-js/modules/_iter-create.js\")(Enumerate, 'Object', function () {\r\n  var that = this;\r\n  var keys = that._k;\r\n  var key;\r\n  do {\r\n    if (that._i >= keys.length) return { value: undefined, done: true };\r\n  } while (!((key = keys[that._i++]) in that._t));\r\n  return { value: key, done: false };\r\n});\r\n\r\n$export($export.S, 'Reflect', {\r\n  enumerate: function enumerate(target) {\r\n    return new Enumerate(target);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.reflect.enumerate.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.reflect.get-own-property-descriptor.js":
/*!*********************************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.reflect.get-own-property-descriptor.js ***!
  \*********************************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 26.1.7 Reflect.getOwnPropertyDescriptor(target, propertyKey)\r\nvar gOPD = __webpack_require__(/*! ./_object-gopd */ \"./node_modules/core-js/modules/_object-gopd.js\");\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\n\r\n$export($export.S, 'Reflect', {\r\n  getOwnPropertyDescriptor: function getOwnPropertyDescriptor(target, propertyKey) {\r\n    return gOPD.f(anObject(target), propertyKey);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.reflect.get-own-property-descriptor.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.reflect.get-prototype-of.js":
/*!**********************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.reflect.get-prototype-of.js ***!
  \**********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 26.1.8 Reflect.getPrototypeOf(target)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar getProto = __webpack_require__(/*! ./_object-gpo */ \"./node_modules/core-js/modules/_object-gpo.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\n\r\n$export($export.S, 'Reflect', {\r\n  getPrototypeOf: function getPrototypeOf(target) {\r\n    return getProto(anObject(target));\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.reflect.get-prototype-of.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.reflect.get.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.reflect.get.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 26.1.6 Reflect.get(target, propertyKey [, receiver])\r\nvar gOPD = __webpack_require__(/*! ./_object-gopd */ \"./node_modules/core-js/modules/_object-gopd.js\");\r\nvar getPrototypeOf = __webpack_require__(/*! ./_object-gpo */ \"./node_modules/core-js/modules/_object-gpo.js\");\r\nvar has = __webpack_require__(/*! ./_has */ \"./node_modules/core-js/modules/_has.js\");\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\n\r\nfunction get(target, propertyKey /* , receiver */) {\r\n  var receiver = arguments.length < 3 ? target : arguments[2];\r\n  var desc, proto;\r\n  if (anObject(target) === receiver) return target[propertyKey];\r\n  if (desc = gOPD.f(target, propertyKey)) return has(desc, 'value')\r\n    ? desc.value\r\n    : desc.get !== undefined\r\n      ? desc.get.call(receiver)\r\n      : undefined;\r\n  if (isObject(proto = getPrototypeOf(target))) return get(proto, propertyKey, receiver);\r\n}\r\n\r\n$export($export.S, 'Reflect', { get: get });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.reflect.get.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.reflect.has.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.reflect.has.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 26.1.9 Reflect.has(target, propertyKey)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Reflect', {\r\n  has: function has(target, propertyKey) {\r\n    return propertyKey in target;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.reflect.has.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.reflect.is-extensible.js":
/*!*******************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.reflect.is-extensible.js ***!
  \*******************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 26.1.10 Reflect.isExtensible(target)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar $isExtensible = Object.isExtensible;\r\n\r\n$export($export.S, 'Reflect', {\r\n  isExtensible: function isExtensible(target) {\r\n    anObject(target);\r\n    return $isExtensible ? $isExtensible(target) : true;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.reflect.is-extensible.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.reflect.own-keys.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.reflect.own-keys.js ***!
  \**************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 26.1.11 Reflect.ownKeys(target)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Reflect', { ownKeys: __webpack_require__(/*! ./_own-keys */ \"./node_modules/core-js/modules/_own-keys.js\") });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.reflect.own-keys.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.reflect.prevent-extensions.js":
/*!************************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.reflect.prevent-extensions.js ***!
  \************************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 26.1.12 Reflect.preventExtensions(target)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar $preventExtensions = Object.preventExtensions;\r\n\r\n$export($export.S, 'Reflect', {\r\n  preventExtensions: function preventExtensions(target) {\r\n    anObject(target);\r\n    try {\r\n      if ($preventExtensions) $preventExtensions(target);\r\n      return true;\r\n    } catch (e) {\r\n      return false;\r\n    }\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.reflect.prevent-extensions.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.reflect.set-prototype-of.js":
/*!**********************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.reflect.set-prototype-of.js ***!
  \**********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 26.1.14 Reflect.setPrototypeOf(target, proto)\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar setProto = __webpack_require__(/*! ./_set-proto */ \"./node_modules/core-js/modules/_set-proto.js\");\r\n\r\nif (setProto) $export($export.S, 'Reflect', {\r\n  setPrototypeOf: function setPrototypeOf(target, proto) {\r\n    setProto.check(target, proto);\r\n    try {\r\n      setProto.set(target, proto);\r\n      return true;\r\n    } catch (e) {\r\n      return false;\r\n    }\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.reflect.set-prototype-of.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.reflect.set.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.reflect.set.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 26.1.13 Reflect.set(target, propertyKey, V [, receiver])\r\nvar dP = __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\");\r\nvar gOPD = __webpack_require__(/*! ./_object-gopd */ \"./node_modules/core-js/modules/_object-gopd.js\");\r\nvar getPrototypeOf = __webpack_require__(/*! ./_object-gpo */ \"./node_modules/core-js/modules/_object-gpo.js\");\r\nvar has = __webpack_require__(/*! ./_has */ \"./node_modules/core-js/modules/_has.js\");\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar createDesc = __webpack_require__(/*! ./_property-desc */ \"./node_modules/core-js/modules/_property-desc.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\n\r\nfunction set(target, propertyKey, V /* , receiver */) {\r\n  var receiver = arguments.length < 4 ? target : arguments[3];\r\n  var ownDesc = gOPD.f(anObject(target), propertyKey);\r\n  var existingDescriptor, proto;\r\n  if (!ownDesc) {\r\n    if (isObject(proto = getPrototypeOf(target))) {\r\n      return set(proto, propertyKey, V, receiver);\r\n    }\r\n    ownDesc = createDesc(0);\r\n  }\r\n  if (has(ownDesc, 'value')) {\r\n    if (ownDesc.writable === false || !isObject(receiver)) return false;\r\n    if (existingDescriptor = gOPD.f(receiver, propertyKey)) {\r\n      if (existingDescriptor.get || existingDescriptor.set || existingDescriptor.writable === false) return false;\r\n      existingDescriptor.value = V;\r\n      dP.f(receiver, propertyKey, existingDescriptor);\r\n    } else dP.f(receiver, propertyKey, createDesc(0, V));\r\n    return true;\r\n  }\r\n  return ownDesc.set === undefined ? false : (ownDesc.set.call(receiver, V), true);\r\n}\r\n\r\n$export($export.S, 'Reflect', { set: set });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.reflect.set.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.regexp.constructor.js":
/*!****************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.regexp.constructor.js ***!
  \****************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar inheritIfRequired = __webpack_require__(/*! ./_inherit-if-required */ \"./node_modules/core-js/modules/_inherit-if-required.js\");\r\nvar dP = __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\").f;\r\nvar gOPN = __webpack_require__(/*! ./_object-gopn */ \"./node_modules/core-js/modules/_object-gopn.js\").f;\r\nvar isRegExp = __webpack_require__(/*! ./_is-regexp */ \"./node_modules/core-js/modules/_is-regexp.js\");\r\nvar $flags = __webpack_require__(/*! ./_flags */ \"./node_modules/core-js/modules/_flags.js\");\r\nvar $RegExp = global.RegExp;\r\nvar Base = $RegExp;\r\nvar proto = $RegExp.prototype;\r\nvar re1 = /a/g;\r\nvar re2 = /a/g;\r\n// \"new\" creates a new object, old webkit buggy here\r\nvar CORRECT_NEW = new $RegExp(re1) !== re1;\r\n\r\nif (__webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\") && (!CORRECT_NEW || __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\")(function () {\r\n  re2[__webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('match')] = false;\r\n  // RegExp constructor can alter flags and IsRegExp works correct with @@match\r\n  return $RegExp(re1) != re1 || $RegExp(re2) == re2 || $RegExp(re1, 'i') != '/a/i';\r\n}))) {\r\n  $RegExp = function RegExp(p, f) {\r\n    var tiRE = this instanceof $RegExp;\r\n    var piRE = isRegExp(p);\r\n    var fiU = f === undefined;\r\n    return !tiRE && piRE && p.constructor === $RegExp && fiU ? p\r\n      : inheritIfRequired(CORRECT_NEW\r\n        ? new Base(piRE && !fiU ? p.source : p, f)\r\n        : Base((piRE = p instanceof $RegExp) ? p.source : p, piRE && fiU ? $flags.call(p) : f)\r\n      , tiRE ? this : proto, $RegExp);\r\n  };\r\n  var proxy = function (key) {\r\n    key in $RegExp || dP($RegExp, key, {\r\n      configurable: true,\r\n      get: function () { return Base[key]; },\r\n      set: function (it) { Base[key] = it; }\r\n    });\r\n  };\r\n  for (var keys = gOPN(Base), i = 0; keys.length > i;) proxy(keys[i++]);\r\n  proto.constructor = $RegExp;\r\n  $RegExp.prototype = proto;\r\n  __webpack_require__(/*! ./_redefine */ \"./node_modules/core-js/modules/_redefine.js\")(global, 'RegExp', $RegExp);\r\n}\r\n\r\n__webpack_require__(/*! ./_set-species */ \"./node_modules/core-js/modules/_set-species.js\")('RegExp');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.regexp.constructor.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.regexp.exec.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.regexp.exec.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar regexpExec = __webpack_require__(/*! ./_regexp-exec */ \"./node_modules/core-js/modules/_regexp-exec.js\");\r\n__webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\")({\r\n  target: 'RegExp',\r\n  proto: true,\r\n  forced: regexpExec !== /./.exec\r\n}, {\r\n  exec: regexpExec\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.regexp.exec.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.regexp.flags.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.regexp.flags.js ***!
  \**********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// 21.2.5.3 get RegExp.prototype.flags()\r\nif (__webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\") && /./g.flags != 'g') __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\").f(RegExp.prototype, 'flags', {\r\n  configurable: true,\r\n  get: __webpack_require__(/*! ./_flags */ \"./node_modules/core-js/modules/_flags.js\")\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.regexp.flags.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.regexp.match.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.regexp.match.js ***!
  \**********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nvar advanceStringIndex = __webpack_require__(/*! ./_advance-string-index */ \"./node_modules/core-js/modules/_advance-string-index.js\");\r\nvar regExpExec = __webpack_require__(/*! ./_regexp-exec-abstract */ \"./node_modules/core-js/modules/_regexp-exec-abstract.js\");\r\n\r\n// @@match logic\r\n__webpack_require__(/*! ./_fix-re-wks */ \"./node_modules/core-js/modules/_fix-re-wks.js\")('match', 1, function (defined, MATCH, $match, maybeCallNative) {\r\n  return [\r\n    // `String.prototype.match` method\r\n    // https://tc39.github.io/ecma262/#sec-string.prototype.match\r\n    function match(regexp) {\r\n      var O = defined(this);\r\n      var fn = regexp == undefined ? undefined : regexp[MATCH];\r\n      return fn !== undefined ? fn.call(regexp, O) : new RegExp(regexp)[MATCH](String(O));\r\n    },\r\n    // `RegExp.prototype[@@match]` method\r\n    // https://tc39.github.io/ecma262/#sec-regexp.prototype-@@match\r\n    function (regexp) {\r\n      var res = maybeCallNative($match, regexp, this);\r\n      if (res.done) return res.value;\r\n      var rx = anObject(regexp);\r\n      var S = String(this);\r\n      if (!rx.global) return regExpExec(rx, S);\r\n      var fullUnicode = rx.unicode;\r\n      rx.lastIndex = 0;\r\n      var A = [];\r\n      var n = 0;\r\n      var result;\r\n      while ((result = regExpExec(rx, S)) !== null) {\r\n        var matchStr = String(result[0]);\r\n        A[n] = matchStr;\r\n        if (matchStr === '') rx.lastIndex = advanceStringIndex(S, toLength(rx.lastIndex), fullUnicode);\r\n        n++;\r\n      }\r\n      return n === 0 ? null : A;\r\n    }\r\n  ];\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.regexp.match.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.regexp.replace.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.regexp.replace.js ***!
  \************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nvar toInteger = __webpack_require__(/*! ./_to-integer */ \"./node_modules/core-js/modules/_to-integer.js\");\r\nvar advanceStringIndex = __webpack_require__(/*! ./_advance-string-index */ \"./node_modules/core-js/modules/_advance-string-index.js\");\r\nvar regExpExec = __webpack_require__(/*! ./_regexp-exec-abstract */ \"./node_modules/core-js/modules/_regexp-exec-abstract.js\");\r\nvar max = Math.max;\r\nvar min = Math.min;\r\nvar floor = Math.floor;\r\nvar SUBSTITUTION_SYMBOLS = /\\$([$&`']|\\d\\d?|<[^>]*>)/g;\r\nvar SUBSTITUTION_SYMBOLS_NO_NAMED = /\\$([$&`']|\\d\\d?)/g;\r\n\r\nvar maybeToString = function (it) {\r\n  return it === undefined ? it : String(it);\r\n};\r\n\r\n// @@replace logic\r\n__webpack_require__(/*! ./_fix-re-wks */ \"./node_modules/core-js/modules/_fix-re-wks.js\")('replace', 2, function (defined, REPLACE, $replace, maybeCallNative) {\r\n  return [\r\n    // `String.prototype.replace` method\r\n    // https://tc39.github.io/ecma262/#sec-string.prototype.replace\r\n    function replace(searchValue, replaceValue) {\r\n      var O = defined(this);\r\n      var fn = searchValue == undefined ? undefined : searchValue[REPLACE];\r\n      return fn !== undefined\r\n        ? fn.call(searchValue, O, replaceValue)\r\n        : $replace.call(String(O), searchValue, replaceValue);\r\n    },\r\n    // `RegExp.prototype[@@replace]` method\r\n    // https://tc39.github.io/ecma262/#sec-regexp.prototype-@@replace\r\n    function (regexp, replaceValue) {\r\n      var res = maybeCallNative($replace, regexp, this, replaceValue);\r\n      if (res.done) return res.value;\r\n\r\n      var rx = anObject(regexp);\r\n      var S = String(this);\r\n      var functionalReplace = typeof replaceValue === 'function';\r\n      if (!functionalReplace) replaceValue = String(replaceValue);\r\n      var global = rx.global;\r\n      if (global) {\r\n        var fullUnicode = rx.unicode;\r\n        rx.lastIndex = 0;\r\n      }\r\n      var results = [];\r\n      while (true) {\r\n        var result = regExpExec(rx, S);\r\n        if (result === null) break;\r\n        results.push(result);\r\n        if (!global) break;\r\n        var matchStr = String(result[0]);\r\n        if (matchStr === '') rx.lastIndex = advanceStringIndex(S, toLength(rx.lastIndex), fullUnicode);\r\n      }\r\n      var accumulatedResult = '';\r\n      var nextSourcePosition = 0;\r\n      for (var i = 0; i < results.length; i++) {\r\n        result = results[i];\r\n        var matched = String(result[0]);\r\n        var position = max(min(toInteger(result.index), S.length), 0);\r\n        var captures = [];\r\n        // NOTE: This is equivalent to\r\n        //   captures = result.slice(1).map(maybeToString)\r\n        // but for some reason `nativeSlice.call(result, 1, result.length)` (called in\r\n        // the slice polyfill when slicing native arrays) \"doesn't work\" in safari 9 and\r\n        // causes a crash (https://pastebin.com/N21QzeQA) when trying to debug it.\r\n        for (var j = 1; j < result.length; j++) captures.push(maybeToString(result[j]));\r\n        var namedCaptures = result.groups;\r\n        if (functionalReplace) {\r\n          var replacerArgs = [matched].concat(captures, position, S);\r\n          if (namedCaptures !== undefined) replacerArgs.push(namedCaptures);\r\n          var replacement = String(replaceValue.apply(undefined, replacerArgs));\r\n        } else {\r\n          replacement = getSubstitution(matched, S, position, captures, namedCaptures, replaceValue);\r\n        }\r\n        if (position >= nextSourcePosition) {\r\n          accumulatedResult += S.slice(nextSourcePosition, position) + replacement;\r\n          nextSourcePosition = position + matched.length;\r\n        }\r\n      }\r\n      return accumulatedResult + S.slice(nextSourcePosition);\r\n    }\r\n  ];\r\n\r\n    // https://tc39.github.io/ecma262/#sec-getsubstitution\r\n  function getSubstitution(matched, str, position, captures, namedCaptures, replacement) {\r\n    var tailPos = position + matched.length;\r\n    var m = captures.length;\r\n    var symbols = SUBSTITUTION_SYMBOLS_NO_NAMED;\r\n    if (namedCaptures !== undefined) {\r\n      namedCaptures = toObject(namedCaptures);\r\n      symbols = SUBSTITUTION_SYMBOLS;\r\n    }\r\n    return $replace.call(replacement, symbols, function (match, ch) {\r\n      var capture;\r\n      switch (ch.charAt(0)) {\r\n        case '$': return '$';\r\n        case '&': return matched;\r\n        case '`': return str.slice(0, position);\r\n        case \"'\": return str.slice(tailPos);\r\n        case '<':\r\n          capture = namedCaptures[ch.slice(1, -1)];\r\n          break;\r\n        default: // \\d\\d?\r\n          var n = +ch;\r\n          if (n === 0) return match;\r\n          if (n > m) {\r\n            var f = floor(n / 10);\r\n            if (f === 0) return match;\r\n            if (f <= m) return captures[f - 1] === undefined ? ch.charAt(1) : captures[f - 1] + ch.charAt(1);\r\n            return match;\r\n          }\r\n          capture = captures[n - 1];\r\n      }\r\n      return capture === undefined ? '' : capture;\r\n    });\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.regexp.replace.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.regexp.search.js":
/*!***********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.regexp.search.js ***!
  \***********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar sameValue = __webpack_require__(/*! ./_same-value */ \"./node_modules/core-js/modules/_same-value.js\");\r\nvar regExpExec = __webpack_require__(/*! ./_regexp-exec-abstract */ \"./node_modules/core-js/modules/_regexp-exec-abstract.js\");\r\n\r\n// @@search logic\r\n__webpack_require__(/*! ./_fix-re-wks */ \"./node_modules/core-js/modules/_fix-re-wks.js\")('search', 1, function (defined, SEARCH, $search, maybeCallNative) {\r\n  return [\r\n    // `String.prototype.search` method\r\n    // https://tc39.github.io/ecma262/#sec-string.prototype.search\r\n    function search(regexp) {\r\n      var O = defined(this);\r\n      var fn = regexp == undefined ? undefined : regexp[SEARCH];\r\n      return fn !== undefined ? fn.call(regexp, O) : new RegExp(regexp)[SEARCH](String(O));\r\n    },\r\n    // `RegExp.prototype[@@search]` method\r\n    // https://tc39.github.io/ecma262/#sec-regexp.prototype-@@search\r\n    function (regexp) {\r\n      var res = maybeCallNative($search, regexp, this);\r\n      if (res.done) return res.value;\r\n      var rx = anObject(regexp);\r\n      var S = String(this);\r\n      var previousLastIndex = rx.lastIndex;\r\n      if (!sameValue(previousLastIndex, 0)) rx.lastIndex = 0;\r\n      var result = regExpExec(rx, S);\r\n      if (!sameValue(rx.lastIndex, previousLastIndex)) rx.lastIndex = previousLastIndex;\r\n      return result === null ? -1 : result.index;\r\n    }\r\n  ];\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.regexp.search.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.regexp.split.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.regexp.split.js ***!
  \**********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n\r\nvar isRegExp = __webpack_require__(/*! ./_is-regexp */ \"./node_modules/core-js/modules/_is-regexp.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar speciesConstructor = __webpack_require__(/*! ./_species-constructor */ \"./node_modules/core-js/modules/_species-constructor.js\");\r\nvar advanceStringIndex = __webpack_require__(/*! ./_advance-string-index */ \"./node_modules/core-js/modules/_advance-string-index.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nvar callRegExpExec = __webpack_require__(/*! ./_regexp-exec-abstract */ \"./node_modules/core-js/modules/_regexp-exec-abstract.js\");\r\nvar regexpExec = __webpack_require__(/*! ./_regexp-exec */ \"./node_modules/core-js/modules/_regexp-exec.js\");\r\nvar fails = __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\");\r\nvar $min = Math.min;\r\nvar $push = [].push;\r\nvar $SPLIT = 'split';\r\nvar LENGTH = 'length';\r\nvar LAST_INDEX = 'lastIndex';\r\nvar MAX_UINT32 = 0xffffffff;\r\n\r\n// babel-minify transpiles RegExp('x', 'y') -> /x/y and it causes SyntaxError\r\nvar SUPPORTS_Y = !fails(function () { RegExp(MAX_UINT32, 'y'); });\r\n\r\n// @@split logic\r\n__webpack_require__(/*! ./_fix-re-wks */ \"./node_modules/core-js/modules/_fix-re-wks.js\")('split', 2, function (defined, SPLIT, $split, maybeCallNative) {\r\n  var internalSplit;\r\n  if (\r\n    'abbc'[$SPLIT](/(b)*/)[1] == 'c' ||\r\n    'test'[$SPLIT](/(?:)/, -1)[LENGTH] != 4 ||\r\n    'ab'[$SPLIT](/(?:ab)*/)[LENGTH] != 2 ||\r\n    '.'[$SPLIT](/(.?)(.?)/)[LENGTH] != 4 ||\r\n    '.'[$SPLIT](/()()/)[LENGTH] > 1 ||\r\n    ''[$SPLIT](/.?/)[LENGTH]\r\n  ) {\r\n    // based on es5-shim implementation, need to rework it\r\n    internalSplit = function (separator, limit) {\r\n      var string = String(this);\r\n      if (separator === undefined && limit === 0) return [];\r\n      // If `separator` is not a regex, use native split\r\n      if (!isRegExp(separator)) return $split.call(string, separator, limit);\r\n      var output = [];\r\n      var flags = (separator.ignoreCase ? 'i' : '') +\r\n                  (separator.multiline ? 'm' : '') +\r\n                  (separator.unicode ? 'u' : '') +\r\n                  (separator.sticky ? 'y' : '');\r\n      var lastLastIndex = 0;\r\n      var splitLimit = limit === undefined ? MAX_UINT32 : limit >>> 0;\r\n      // Make `global` and avoid `lastIndex` issues by working with a copy\r\n      var separatorCopy = new RegExp(separator.source, flags + 'g');\r\n      var match, lastIndex, lastLength;\r\n      while (match = regexpExec.call(separatorCopy, string)) {\r\n        lastIndex = separatorCopy[LAST_INDEX];\r\n        if (lastIndex > lastLastIndex) {\r\n          output.push(string.slice(lastLastIndex, match.index));\r\n          if (match[LENGTH] > 1 && match.index < string[LENGTH]) $push.apply(output, match.slice(1));\r\n          lastLength = match[0][LENGTH];\r\n          lastLastIndex = lastIndex;\r\n          if (output[LENGTH] >= splitLimit) break;\r\n        }\r\n        if (separatorCopy[LAST_INDEX] === match.index) separatorCopy[LAST_INDEX]++; // Avoid an infinite loop\r\n      }\r\n      if (lastLastIndex === string[LENGTH]) {\r\n        if (lastLength || !separatorCopy.test('')) output.push('');\r\n      } else output.push(string.slice(lastLastIndex));\r\n      return output[LENGTH] > splitLimit ? output.slice(0, splitLimit) : output;\r\n    };\r\n  // Chakra, V8\r\n  } else if ('0'[$SPLIT](undefined, 0)[LENGTH]) {\r\n    internalSplit = function (separator, limit) {\r\n      return separator === undefined && limit === 0 ? [] : $split.call(this, separator, limit);\r\n    };\r\n  } else {\r\n    internalSplit = $split;\r\n  }\r\n\r\n  return [\r\n    // `String.prototype.split` method\r\n    // https://tc39.github.io/ecma262/#sec-string.prototype.split\r\n    function split(separator, limit) {\r\n      var O = defined(this);\r\n      var splitter = separator == undefined ? undefined : separator[SPLIT];\r\n      return splitter !== undefined\r\n        ? splitter.call(separator, O, limit)\r\n        : internalSplit.call(String(O), separator, limit);\r\n    },\r\n    // `RegExp.prototype[@@split]` method\r\n    // https://tc39.github.io/ecma262/#sec-regexp.prototype-@@split\r\n    //\r\n    // NOTE: This cannot be properly polyfilled in engines that don't support\r\n    // the 'y' flag.\r\n    function (regexp, limit) {\r\n      var res = maybeCallNative(internalSplit, regexp, this, limit, internalSplit !== $split);\r\n      if (res.done) return res.value;\r\n\r\n      var rx = anObject(regexp);\r\n      var S = String(this);\r\n      var C = speciesConstructor(rx, RegExp);\r\n\r\n      var unicodeMatching = rx.unicode;\r\n      var flags = (rx.ignoreCase ? 'i' : '') +\r\n                  (rx.multiline ? 'm' : '') +\r\n                  (rx.unicode ? 'u' : '') +\r\n                  (SUPPORTS_Y ? 'y' : 'g');\r\n\r\n      // ^(? + rx + ) is needed, in combination with some S slicing, to\r\n      // simulate the 'y' flag.\r\n      var splitter = new C(SUPPORTS_Y ? rx : '^(?:' + rx.source + ')', flags);\r\n      var lim = limit === undefined ? MAX_UINT32 : limit >>> 0;\r\n      if (lim === 0) return [];\r\n      if (S.length === 0) return callRegExpExec(splitter, S) === null ? [S] : [];\r\n      var p = 0;\r\n      var q = 0;\r\n      var A = [];\r\n      while (q < S.length) {\r\n        splitter.lastIndex = SUPPORTS_Y ? q : 0;\r\n        var z = callRegExpExec(splitter, SUPPORTS_Y ? S : S.slice(q));\r\n        var e;\r\n        if (\r\n          z === null ||\r\n          (e = $min(toLength(splitter.lastIndex + (SUPPORTS_Y ? 0 : q)), S.length)) === p\r\n        ) {\r\n          q = advanceStringIndex(S, q, unicodeMatching);\r\n        } else {\r\n          A.push(S.slice(p, q));\r\n          if (A.length === lim) return A;\r\n          for (var i = 1; i <= z.length - 1; i++) {\r\n            A.push(z[i]);\r\n            if (A.length === lim) return A;\r\n          }\r\n          q = p = e;\r\n        }\r\n      }\r\n      A.push(S.slice(p));\r\n      return A;\r\n    }\r\n  ];\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.regexp.split.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.regexp.to-string.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.regexp.to-string.js ***!
  \**************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n__webpack_require__(/*! ./es6.regexp.flags */ \"./node_modules/core-js/modules/es6.regexp.flags.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar $flags = __webpack_require__(/*! ./_flags */ \"./node_modules/core-js/modules/_flags.js\");\r\nvar DESCRIPTORS = __webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\");\r\nvar TO_STRING = 'toString';\r\nvar $toString = /./[TO_STRING];\r\n\r\nvar define = function (fn) {\r\n  __webpack_require__(/*! ./_redefine */ \"./node_modules/core-js/modules/_redefine.js\")(RegExp.prototype, TO_STRING, fn, true);\r\n};\r\n\r\n// 21.2.5.14 RegExp.prototype.toString()\r\nif (__webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\")(function () { return $toString.call({ source: 'a', flags: 'b' }) != '/a/b'; })) {\r\n  define(function toString() {\r\n    var R = anObject(this);\r\n    return '/'.concat(R.source, '/',\r\n      'flags' in R ? R.flags : !DESCRIPTORS && R instanceof RegExp ? $flags.call(R) : undefined);\r\n  });\r\n// FF44- RegExp#toString has a wrong name\r\n} else if ($toString.name != TO_STRING) {\r\n  define(function toString() {\r\n    return $toString.call(this);\r\n  });\r\n}\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.regexp.to-string.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.set.js":
/*!*************************************************!*\
  !*** ./node_modules/core-js/modules/es6.set.js ***!
  \*************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar strong = __webpack_require__(/*! ./_collection-strong */ \"./node_modules/core-js/modules/_collection-strong.js\");\r\nvar validate = __webpack_require__(/*! ./_validate-collection */ \"./node_modules/core-js/modules/_validate-collection.js\");\r\nvar SET = 'Set';\r\n\r\n// 23.2 Set Objects\r\nmodule.exports = __webpack_require__(/*! ./_collection */ \"./node_modules/core-js/modules/_collection.js\")(SET, function (get) {\r\n  return function Set() { return get(this, arguments.length > 0 ? arguments[0] : undefined); };\r\n}, {\r\n  // 23.2.3.1 Set.prototype.add(value)\r\n  add: function add(value) {\r\n    return strong.def(validate(this, SET), value = value === 0 ? 0 : value, value);\r\n  }\r\n}, strong);\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.set.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.anchor.js":
/*!***********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.anchor.js ***!
  \***********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// B.2.3.2 String.prototype.anchor(name)\r\n__webpack_require__(/*! ./_string-html */ \"./node_modules/core-js/modules/_string-html.js\")('anchor', function (createHTML) {\r\n  return function anchor(name) {\r\n    return createHTML(this, 'a', 'name', name);\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.anchor.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.big.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.big.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// B.2.3.3 String.prototype.big()\r\n__webpack_require__(/*! ./_string-html */ \"./node_modules/core-js/modules/_string-html.js\")('big', function (createHTML) {\r\n  return function big() {\r\n    return createHTML(this, 'big', '', '');\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.big.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.blink.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.blink.js ***!
  \**********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// B.2.3.4 String.prototype.blink()\r\n__webpack_require__(/*! ./_string-html */ \"./node_modules/core-js/modules/_string-html.js\")('blink', function (createHTML) {\r\n  return function blink() {\r\n    return createHTML(this, 'blink', '', '');\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.blink.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.bold.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.bold.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// B.2.3.5 String.prototype.bold()\r\n__webpack_require__(/*! ./_string-html */ \"./node_modules/core-js/modules/_string-html.js\")('bold', function (createHTML) {\r\n  return function bold() {\r\n    return createHTML(this, 'b', '', '');\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.bold.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.code-point-at.js":
/*!******************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.code-point-at.js ***!
  \******************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $at = __webpack_require__(/*! ./_string-at */ \"./node_modules/core-js/modules/_string-at.js\")(false);\r\n$export($export.P, 'String', {\r\n  // 21.1.3.3 String.prototype.codePointAt(pos)\r\n  codePointAt: function codePointAt(pos) {\r\n    return $at(this, pos);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.code-point-at.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.ends-with.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.ends-with.js ***!
  \**************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("// 21.1.3.6 String.prototype.endsWith(searchString [, endPosition])\r\n\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nvar context = __webpack_require__(/*! ./_string-context */ \"./node_modules/core-js/modules/_string-context.js\");\r\nvar ENDS_WITH = 'endsWith';\r\nvar $endsWith = ''[ENDS_WITH];\r\n\r\n$export($export.P + $export.F * __webpack_require__(/*! ./_fails-is-regexp */ \"./node_modules/core-js/modules/_fails-is-regexp.js\")(ENDS_WITH), 'String', {\r\n  endsWith: function endsWith(searchString /* , endPosition = @length */) {\r\n    var that = context(this, searchString, ENDS_WITH);\r\n    var endPosition = arguments.length > 1 ? arguments[1] : undefined;\r\n    var len = toLength(that.length);\r\n    var end = endPosition === undefined ? len : Math.min(toLength(endPosition), len);\r\n    var search = String(searchString);\r\n    return $endsWith\r\n      ? $endsWith.call(that, search, end)\r\n      : that.slice(end - search.length, end) === search;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.ends-with.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.fixed.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.fixed.js ***!
  \**********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// B.2.3.6 String.prototype.fixed()\r\n__webpack_require__(/*! ./_string-html */ \"./node_modules/core-js/modules/_string-html.js\")('fixed', function (createHTML) {\r\n  return function fixed() {\r\n    return createHTML(this, 'tt', '', '');\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.fixed.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.fontcolor.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.fontcolor.js ***!
  \**************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// B.2.3.7 String.prototype.fontcolor(color)\r\n__webpack_require__(/*! ./_string-html */ \"./node_modules/core-js/modules/_string-html.js\")('fontcolor', function (createHTML) {\r\n  return function fontcolor(color) {\r\n    return createHTML(this, 'font', 'color', color);\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.fontcolor.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.fontsize.js":
/*!*************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.fontsize.js ***!
  \*************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// B.2.3.8 String.prototype.fontsize(size)\r\n__webpack_require__(/*! ./_string-html */ \"./node_modules/core-js/modules/_string-html.js\")('fontsize', function (createHTML) {\r\n  return function fontsize(size) {\r\n    return createHTML(this, 'font', 'size', size);\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.fontsize.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.from-code-point.js":
/*!********************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.from-code-point.js ***!
  \********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar toAbsoluteIndex = __webpack_require__(/*! ./_to-absolute-index */ \"./node_modules/core-js/modules/_to-absolute-index.js\");\r\nvar fromCharCode = String.fromCharCode;\r\nvar $fromCodePoint = String.fromCodePoint;\r\n\r\n// length should be 1, old FF problem\r\n$export($export.S + $export.F * (!!$fromCodePoint && $fromCodePoint.length != 1), 'String', {\r\n  // 21.1.2.2 String.fromCodePoint(...codePoints)\r\n  fromCodePoint: function fromCodePoint(x) { // eslint-disable-line no-unused-vars\r\n    var res = [];\r\n    var aLen = arguments.length;\r\n    var i = 0;\r\n    var code;\r\n    while (aLen > i) {\r\n      code = +arguments[i++];\r\n      if (toAbsoluteIndex(code, 0x10ffff) !== code) throw RangeError(code + ' is not a valid code point');\r\n      res.push(code < 0x10000\r\n        ? fromCharCode(code)\r\n        : fromCharCode(((code -= 0x10000) >> 10) + 0xd800, code % 0x400 + 0xdc00)\r\n      );\r\n    } return res.join('');\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.from-code-point.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.includes.js":
/*!*************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.includes.js ***!
  \*************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("// 21.1.3.7 String.prototype.includes(searchString, position = 0)\r\n\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar context = __webpack_require__(/*! ./_string-context */ \"./node_modules/core-js/modules/_string-context.js\");\r\nvar INCLUDES = 'includes';\r\n\r\n$export($export.P + $export.F * __webpack_require__(/*! ./_fails-is-regexp */ \"./node_modules/core-js/modules/_fails-is-regexp.js\")(INCLUDES), 'String', {\r\n  includes: function includes(searchString /* , position = 0 */) {\r\n    return !!~context(this, searchString, INCLUDES)\r\n      .indexOf(searchString, arguments.length > 1 ? arguments[1] : undefined);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.includes.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.italics.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.italics.js ***!
  \************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// B.2.3.9 String.prototype.italics()\r\n__webpack_require__(/*! ./_string-html */ \"./node_modules/core-js/modules/_string-html.js\")('italics', function (createHTML) {\r\n  return function italics() {\r\n    return createHTML(this, 'i', '', '');\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.italics.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.iterator.js":
/*!*************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.iterator.js ***!
  \*************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $at = __webpack_require__(/*! ./_string-at */ \"./node_modules/core-js/modules/_string-at.js\")(true);\r\n\r\n// 21.1.3.27 String.prototype[@@iterator]()\r\n__webpack_require__(/*! ./_iter-define */ \"./node_modules/core-js/modules/_iter-define.js\")(String, 'String', function (iterated) {\r\n  this._t = String(iterated); // target\r\n  this._i = 0;                // next index\r\n// 21.1.5.2.1 %StringIteratorPrototype%.next()\r\n}, function () {\r\n  var O = this._t;\r\n  var index = this._i;\r\n  var point;\r\n  if (index >= O.length) return { value: undefined, done: true };\r\n  point = $at(O, index);\r\n  this._i += point.length;\r\n  return { value: point, done: false };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.iterator.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.link.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.link.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// B.2.3.10 String.prototype.link(url)\r\n__webpack_require__(/*! ./_string-html */ \"./node_modules/core-js/modules/_string-html.js\")('link', function (createHTML) {\r\n  return function link(url) {\r\n    return createHTML(this, 'a', 'href', url);\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.link.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.raw.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.raw.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar toIObject = __webpack_require__(/*! ./_to-iobject */ \"./node_modules/core-js/modules/_to-iobject.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\n\r\n$export($export.S, 'String', {\r\n  // 21.1.2.4 String.raw(callSite, ...substitutions)\r\n  raw: function raw(callSite) {\r\n    var tpl = toIObject(callSite.raw);\r\n    var len = toLength(tpl.length);\r\n    var aLen = arguments.length;\r\n    var res = [];\r\n    var i = 0;\r\n    while (len > i) {\r\n      res.push(String(tpl[i++]));\r\n      if (i < aLen) res.push(String(arguments[i]));\r\n    } return res.join('');\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.raw.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.repeat.js":
/*!***********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.repeat.js ***!
  \***********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.P, 'String', {\r\n  // 21.1.3.13 String.prototype.repeat(count)\r\n  repeat: __webpack_require__(/*! ./_string-repeat */ \"./node_modules/core-js/modules/_string-repeat.js\")\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.repeat.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.small.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.small.js ***!
  \**********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// B.2.3.11 String.prototype.small()\r\n__webpack_require__(/*! ./_string-html */ \"./node_modules/core-js/modules/_string-html.js\")('small', function (createHTML) {\r\n  return function small() {\r\n    return createHTML(this, 'small', '', '');\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.small.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.starts-with.js":
/*!****************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.starts-with.js ***!
  \****************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("// 21.1.3.18 String.prototype.startsWith(searchString [, position ])\r\n\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nvar context = __webpack_require__(/*! ./_string-context */ \"./node_modules/core-js/modules/_string-context.js\");\r\nvar STARTS_WITH = 'startsWith';\r\nvar $startsWith = ''[STARTS_WITH];\r\n\r\n$export($export.P + $export.F * __webpack_require__(/*! ./_fails-is-regexp */ \"./node_modules/core-js/modules/_fails-is-regexp.js\")(STARTS_WITH), 'String', {\r\n  startsWith: function startsWith(searchString /* , position = 0 */) {\r\n    var that = context(this, searchString, STARTS_WITH);\r\n    var index = toLength(Math.min(arguments.length > 1 ? arguments[1] : undefined, that.length));\r\n    var search = String(searchString);\r\n    return $startsWith\r\n      ? $startsWith.call(that, search, index)\r\n      : that.slice(index, index + search.length) === search;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.starts-with.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.strike.js":
/*!***********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.strike.js ***!
  \***********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// B.2.3.12 String.prototype.strike()\r\n__webpack_require__(/*! ./_string-html */ \"./node_modules/core-js/modules/_string-html.js\")('strike', function (createHTML) {\r\n  return function strike() {\r\n    return createHTML(this, 'strike', '', '');\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.strike.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.sub.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.sub.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// B.2.3.13 String.prototype.sub()\r\n__webpack_require__(/*! ./_string-html */ \"./node_modules/core-js/modules/_string-html.js\")('sub', function (createHTML) {\r\n  return function sub() {\r\n    return createHTML(this, 'sub', '', '');\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.sub.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.sup.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.sup.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// B.2.3.14 String.prototype.sup()\r\n__webpack_require__(/*! ./_string-html */ \"./node_modules/core-js/modules/_string-html.js\")('sup', function (createHTML) {\r\n  return function sup() {\r\n    return createHTML(this, 'sup', '', '');\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.sup.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.string.trim.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es6.string.trim.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// 21.1.3.25 String.prototype.trim()\r\n__webpack_require__(/*! ./_string-trim */ \"./node_modules/core-js/modules/_string-trim.js\")('trim', function ($trim) {\r\n  return function trim() {\r\n    return $trim(this, 3);\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.string.trim.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.symbol.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/es6.symbol.js ***!
  \****************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// ECMAScript 6 symbols shim\r\nvar global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar has = __webpack_require__(/*! ./_has */ \"./node_modules/core-js/modules/_has.js\");\r\nvar DESCRIPTORS = __webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\");\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar redefine = __webpack_require__(/*! ./_redefine */ \"./node_modules/core-js/modules/_redefine.js\");\r\nvar META = __webpack_require__(/*! ./_meta */ \"./node_modules/core-js/modules/_meta.js\").KEY;\r\nvar $fails = __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\");\r\nvar shared = __webpack_require__(/*! ./_shared */ \"./node_modules/core-js/modules/_shared.js\");\r\nvar setToStringTag = __webpack_require__(/*! ./_set-to-string-tag */ \"./node_modules/core-js/modules/_set-to-string-tag.js\");\r\nvar uid = __webpack_require__(/*! ./_uid */ \"./node_modules/core-js/modules/_uid.js\");\r\nvar wks = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\");\r\nvar wksExt = __webpack_require__(/*! ./_wks-ext */ \"./node_modules/core-js/modules/_wks-ext.js\");\r\nvar wksDefine = __webpack_require__(/*! ./_wks-define */ \"./node_modules/core-js/modules/_wks-define.js\");\r\nvar enumKeys = __webpack_require__(/*! ./_enum-keys */ \"./node_modules/core-js/modules/_enum-keys.js\");\r\nvar isArray = __webpack_require__(/*! ./_is-array */ \"./node_modules/core-js/modules/_is-array.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar toIObject = __webpack_require__(/*! ./_to-iobject */ \"./node_modules/core-js/modules/_to-iobject.js\");\r\nvar toPrimitive = __webpack_require__(/*! ./_to-primitive */ \"./node_modules/core-js/modules/_to-primitive.js\");\r\nvar createDesc = __webpack_require__(/*! ./_property-desc */ \"./node_modules/core-js/modules/_property-desc.js\");\r\nvar _create = __webpack_require__(/*! ./_object-create */ \"./node_modules/core-js/modules/_object-create.js\");\r\nvar gOPNExt = __webpack_require__(/*! ./_object-gopn-ext */ \"./node_modules/core-js/modules/_object-gopn-ext.js\");\r\nvar $GOPD = __webpack_require__(/*! ./_object-gopd */ \"./node_modules/core-js/modules/_object-gopd.js\");\r\nvar $GOPS = __webpack_require__(/*! ./_object-gops */ \"./node_modules/core-js/modules/_object-gops.js\");\r\nvar $DP = __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\");\r\nvar $keys = __webpack_require__(/*! ./_object-keys */ \"./node_modules/core-js/modules/_object-keys.js\");\r\nvar gOPD = $GOPD.f;\r\nvar dP = $DP.f;\r\nvar gOPN = gOPNExt.f;\r\nvar $Symbol = global.Symbol;\r\nvar $JSON = global.JSON;\r\nvar _stringify = $JSON && $JSON.stringify;\r\nvar PROTOTYPE = 'prototype';\r\nvar HIDDEN = wks('_hidden');\r\nvar TO_PRIMITIVE = wks('toPrimitive');\r\nvar isEnum = {}.propertyIsEnumerable;\r\nvar SymbolRegistry = shared('symbol-registry');\r\nvar AllSymbols = shared('symbols');\r\nvar OPSymbols = shared('op-symbols');\r\nvar ObjectProto = Object[PROTOTYPE];\r\nvar USE_NATIVE = typeof $Symbol == 'function' && !!$GOPS.f;\r\nvar QObject = global.QObject;\r\n// Don't use setters in Qt Script, https://github.com/zloirock/core-js/issues/173\r\nvar setter = !QObject || !QObject[PROTOTYPE] || !QObject[PROTOTYPE].findChild;\r\n\r\n// fallback for old Android, https://code.google.com/p/v8/issues/detail?id=687\r\nvar setSymbolDesc = DESCRIPTORS && $fails(function () {\r\n  return _create(dP({}, 'a', {\r\n    get: function () { return dP(this, 'a', { value: 7 }).a; }\r\n  })).a != 7;\r\n}) ? function (it, key, D) {\r\n  var protoDesc = gOPD(ObjectProto, key);\r\n  if (protoDesc) delete ObjectProto[key];\r\n  dP(it, key, D);\r\n  if (protoDesc && it !== ObjectProto) dP(ObjectProto, key, protoDesc);\r\n} : dP;\r\n\r\nvar wrap = function (tag) {\r\n  var sym = AllSymbols[tag] = _create($Symbol[PROTOTYPE]);\r\n  sym._k = tag;\r\n  return sym;\r\n};\r\n\r\nvar isSymbol = USE_NATIVE && typeof $Symbol.iterator == 'symbol' ? function (it) {\r\n  return typeof it == 'symbol';\r\n} : function (it) {\r\n  return it instanceof $Symbol;\r\n};\r\n\r\nvar $defineProperty = function defineProperty(it, key, D) {\r\n  if (it === ObjectProto) $defineProperty(OPSymbols, key, D);\r\n  anObject(it);\r\n  key = toPrimitive(key, true);\r\n  anObject(D);\r\n  if (has(AllSymbols, key)) {\r\n    if (!D.enumerable) {\r\n      if (!has(it, HIDDEN)) dP(it, HIDDEN, createDesc(1, {}));\r\n      it[HIDDEN][key] = true;\r\n    } else {\r\n      if (has(it, HIDDEN) && it[HIDDEN][key]) it[HIDDEN][key] = false;\r\n      D = _create(D, { enumerable: createDesc(0, false) });\r\n    } return setSymbolDesc(it, key, D);\r\n  } return dP(it, key, D);\r\n};\r\nvar $defineProperties = function defineProperties(it, P) {\r\n  anObject(it);\r\n  var keys = enumKeys(P = toIObject(P));\r\n  var i = 0;\r\n  var l = keys.length;\r\n  var key;\r\n  while (l > i) $defineProperty(it, key = keys[i++], P[key]);\r\n  return it;\r\n};\r\nvar $create = function create(it, P) {\r\n  return P === undefined ? _create(it) : $defineProperties(_create(it), P);\r\n};\r\nvar $propertyIsEnumerable = function propertyIsEnumerable(key) {\r\n  var E = isEnum.call(this, key = toPrimitive(key, true));\r\n  if (this === ObjectProto && has(AllSymbols, key) && !has(OPSymbols, key)) return false;\r\n  return E || !has(this, key) || !has(AllSymbols, key) || has(this, HIDDEN) && this[HIDDEN][key] ? E : true;\r\n};\r\nvar $getOwnPropertyDescriptor = function getOwnPropertyDescriptor(it, key) {\r\n  it = toIObject(it);\r\n  key = toPrimitive(key, true);\r\n  if (it === ObjectProto && has(AllSymbols, key) && !has(OPSymbols, key)) return;\r\n  var D = gOPD(it, key);\r\n  if (D && has(AllSymbols, key) && !(has(it, HIDDEN) && it[HIDDEN][key])) D.enumerable = true;\r\n  return D;\r\n};\r\nvar $getOwnPropertyNames = function getOwnPropertyNames(it) {\r\n  var names = gOPN(toIObject(it));\r\n  var result = [];\r\n  var i = 0;\r\n  var key;\r\n  while (names.length > i) {\r\n    if (!has(AllSymbols, key = names[i++]) && key != HIDDEN && key != META) result.push(key);\r\n  } return result;\r\n};\r\nvar $getOwnPropertySymbols = function getOwnPropertySymbols(it) {\r\n  var IS_OP = it === ObjectProto;\r\n  var names = gOPN(IS_OP ? OPSymbols : toIObject(it));\r\n  var result = [];\r\n  var i = 0;\r\n  var key;\r\n  while (names.length > i) {\r\n    if (has(AllSymbols, key = names[i++]) && (IS_OP ? has(ObjectProto, key) : true)) result.push(AllSymbols[key]);\r\n  } return result;\r\n};\r\n\r\n// 19.4.1.1 Symbol([description])\r\nif (!USE_NATIVE) {\r\n  $Symbol = function Symbol() {\r\n    if (this instanceof $Symbol) throw TypeError('Symbol is not a constructor!');\r\n    var tag = uid(arguments.length > 0 ? arguments[0] : undefined);\r\n    var $set = function (value) {\r\n      if (this === ObjectProto) $set.call(OPSymbols, value);\r\n      if (has(this, HIDDEN) && has(this[HIDDEN], tag)) this[HIDDEN][tag] = false;\r\n      setSymbolDesc(this, tag, createDesc(1, value));\r\n    };\r\n    if (DESCRIPTORS && setter) setSymbolDesc(ObjectProto, tag, { configurable: true, set: $set });\r\n    return wrap(tag);\r\n  };\r\n  redefine($Symbol[PROTOTYPE], 'toString', function toString() {\r\n    return this._k;\r\n  });\r\n\r\n  $GOPD.f = $getOwnPropertyDescriptor;\r\n  $DP.f = $defineProperty;\r\n  __webpack_require__(/*! ./_object-gopn */ \"./node_modules/core-js/modules/_object-gopn.js\").f = gOPNExt.f = $getOwnPropertyNames;\r\n  __webpack_require__(/*! ./_object-pie */ \"./node_modules/core-js/modules/_object-pie.js\").f = $propertyIsEnumerable;\r\n  $GOPS.f = $getOwnPropertySymbols;\r\n\r\n  if (DESCRIPTORS && !__webpack_require__(/*! ./_library */ \"./node_modules/core-js/modules/_library.js\")) {\r\n    redefine(ObjectProto, 'propertyIsEnumerable', $propertyIsEnumerable, true);\r\n  }\r\n\r\n  wksExt.f = function (name) {\r\n    return wrap(wks(name));\r\n  };\r\n}\r\n\r\n$export($export.G + $export.W + $export.F * !USE_NATIVE, { Symbol: $Symbol });\r\n\r\nfor (var es6Symbols = (\r\n  // 19.4.2.2, 19.4.2.3, 19.4.2.4, 19.4.2.6, 19.4.2.8, 19.4.2.9, 19.4.2.10, 19.4.2.11, 19.4.2.12, 19.4.2.13, 19.4.2.14\r\n  'hasInstance,isConcatSpreadable,iterator,match,replace,search,species,split,toPrimitive,toStringTag,unscopables'\r\n).split(','), j = 0; es6Symbols.length > j;)wks(es6Symbols[j++]);\r\n\r\nfor (var wellKnownSymbols = $keys(wks.store), k = 0; wellKnownSymbols.length > k;) wksDefine(wellKnownSymbols[k++]);\r\n\r\n$export($export.S + $export.F * !USE_NATIVE, 'Symbol', {\r\n  // 19.4.2.1 Symbol.for(key)\r\n  'for': function (key) {\r\n    return has(SymbolRegistry, key += '')\r\n      ? SymbolRegistry[key]\r\n      : SymbolRegistry[key] = $Symbol(key);\r\n  },\r\n  // 19.4.2.5 Symbol.keyFor(sym)\r\n  keyFor: function keyFor(sym) {\r\n    if (!isSymbol(sym)) throw TypeError(sym + ' is not a symbol!');\r\n    for (var key in SymbolRegistry) if (SymbolRegistry[key] === sym) return key;\r\n  },\r\n  useSetter: function () { setter = true; },\r\n  useSimple: function () { setter = false; }\r\n});\r\n\r\n$export($export.S + $export.F * !USE_NATIVE, 'Object', {\r\n  // 19.1.2.2 Object.create(O [, Properties])\r\n  create: $create,\r\n  // 19.1.2.4 Object.defineProperty(O, P, Attributes)\r\n  defineProperty: $defineProperty,\r\n  // 19.1.2.3 Object.defineProperties(O, Properties)\r\n  defineProperties: $defineProperties,\r\n  // 19.1.2.6 Object.getOwnPropertyDescriptor(O, P)\r\n  getOwnPropertyDescriptor: $getOwnPropertyDescriptor,\r\n  // 19.1.2.7 Object.getOwnPropertyNames(O)\r\n  getOwnPropertyNames: $getOwnPropertyNames,\r\n  // 19.1.2.8 Object.getOwnPropertySymbols(O)\r\n  getOwnPropertySymbols: $getOwnPropertySymbols\r\n});\r\n\r\n// Chrome 38 and 39 `Object.getOwnPropertySymbols` fails on primitives\r\n// https://bugs.chromium.org/p/v8/issues/detail?id=3443\r\nvar FAILS_ON_PRIMITIVES = $fails(function () { $GOPS.f(1); });\r\n\r\n$export($export.S + $export.F * FAILS_ON_PRIMITIVES, 'Object', {\r\n  getOwnPropertySymbols: function getOwnPropertySymbols(it) {\r\n    return $GOPS.f(toObject(it));\r\n  }\r\n});\r\n\r\n// 24.3.2 JSON.stringify(value [, replacer [, space]])\r\n$JSON && $export($export.S + $export.F * (!USE_NATIVE || $fails(function () {\r\n  var S = $Symbol();\r\n  // MS Edge converts symbol values to JSON as {}\r\n  // WebKit converts symbol values to JSON as null\r\n  // V8 throws on boxed symbols\r\n  return _stringify([S]) != '[null]' || _stringify({ a: S }) != '{}' || _stringify(Object(S)) != '{}';\r\n})), 'JSON', {\r\n  stringify: function stringify(it) {\r\n    var args = [it];\r\n    var i = 1;\r\n    var replacer, $replacer;\r\n    while (arguments.length > i) args.push(arguments[i++]);\r\n    $replacer = replacer = args[1];\r\n    if (!isObject(replacer) && it === undefined || isSymbol(it)) return; // IE8 returns string on undefined\r\n    if (!isArray(replacer)) replacer = function (key, value) {\r\n      if (typeof $replacer == 'function') value = $replacer.call(this, key, value);\r\n      if (!isSymbol(value)) return value;\r\n    };\r\n    args[1] = replacer;\r\n    return _stringify.apply($JSON, args);\r\n  }\r\n});\r\n\r\n// 19.4.3.4 Symbol.prototype[@@toPrimitive](hint)\r\n$Symbol[PROTOTYPE][TO_PRIMITIVE] || __webpack_require__(/*! ./_hide */ \"./node_modules/core-js/modules/_hide.js\")($Symbol[PROTOTYPE], TO_PRIMITIVE, $Symbol[PROTOTYPE].valueOf);\r\n// 19.4.3.5 Symbol.prototype[@@toStringTag]\r\nsetToStringTag($Symbol, 'Symbol');\r\n// 20.2.1.9 Math[@@toStringTag]\r\nsetToStringTag(Math, 'Math', true);\r\n// 24.3.3 JSON[@@toStringTag]\r\nsetToStringTag(global.JSON, 'JSON', true);\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.symbol.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.typed.array-buffer.js":
/*!****************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.typed.array-buffer.js ***!
  \****************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $typed = __webpack_require__(/*! ./_typed */ \"./node_modules/core-js/modules/_typed.js\");\r\nvar buffer = __webpack_require__(/*! ./_typed-buffer */ \"./node_modules/core-js/modules/_typed-buffer.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar toAbsoluteIndex = __webpack_require__(/*! ./_to-absolute-index */ \"./node_modules/core-js/modules/_to-absolute-index.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar ArrayBuffer = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\").ArrayBuffer;\r\nvar speciesConstructor = __webpack_require__(/*! ./_species-constructor */ \"./node_modules/core-js/modules/_species-constructor.js\");\r\nvar $ArrayBuffer = buffer.ArrayBuffer;\r\nvar $DataView = buffer.DataView;\r\nvar $isView = $typed.ABV && ArrayBuffer.isView;\r\nvar $slice = $ArrayBuffer.prototype.slice;\r\nvar VIEW = $typed.VIEW;\r\nvar ARRAY_BUFFER = 'ArrayBuffer';\r\n\r\n$export($export.G + $export.W + $export.F * (ArrayBuffer !== $ArrayBuffer), { ArrayBuffer: $ArrayBuffer });\r\n\r\n$export($export.S + $export.F * !$typed.CONSTR, ARRAY_BUFFER, {\r\n  // 24.1.3.1 ArrayBuffer.isView(arg)\r\n  isView: function isView(it) {\r\n    return $isView && $isView(it) || isObject(it) && VIEW in it;\r\n  }\r\n});\r\n\r\n$export($export.P + $export.U + $export.F * __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\")(function () {\r\n  return !new $ArrayBuffer(2).slice(1, undefined).byteLength;\r\n}), ARRAY_BUFFER, {\r\n  // 24.1.4.3 ArrayBuffer.prototype.slice(start, end)\r\n  slice: function slice(start, end) {\r\n    if ($slice !== undefined && end === undefined) return $slice.call(anObject(this), start); // FF fix\r\n    var len = anObject(this).byteLength;\r\n    var first = toAbsoluteIndex(start, len);\r\n    var fin = toAbsoluteIndex(end === undefined ? len : end, len);\r\n    var result = new (speciesConstructor(this, $ArrayBuffer))(toLength(fin - first));\r\n    var viewS = new $DataView(this);\r\n    var viewT = new $DataView(result);\r\n    var index = 0;\r\n    while (first < fin) {\r\n      viewT.setUint8(index++, viewS.getUint8(first++));\r\n    } return result;\r\n  }\r\n});\r\n\r\n__webpack_require__(/*! ./_set-species */ \"./node_modules/core-js/modules/_set-species.js\")(ARRAY_BUFFER);\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.typed.array-buffer.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.typed.data-view.js":
/*!*************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.typed.data-view.js ***!
  \*************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n$export($export.G + $export.W + $export.F * !__webpack_require__(/*! ./_typed */ \"./node_modules/core-js/modules/_typed.js\").ABV, {\r\n  DataView: __webpack_require__(/*! ./_typed-buffer */ \"./node_modules/core-js/modules/_typed-buffer.js\").DataView\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.typed.data-view.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.typed.float32-array.js":
/*!*****************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.typed.float32-array.js ***!
  \*****************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("__webpack_require__(/*! ./_typed-array */ \"./node_modules/core-js/modules/_typed-array.js\")('Float32', 4, function (init) {\r\n  return function Float32Array(data, byteOffset, length) {\r\n    return init(this, data, byteOffset, length);\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.typed.float32-array.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.typed.float64-array.js":
/*!*****************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.typed.float64-array.js ***!
  \*****************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("__webpack_require__(/*! ./_typed-array */ \"./node_modules/core-js/modules/_typed-array.js\")('Float64', 8, function (init) {\r\n  return function Float64Array(data, byteOffset, length) {\r\n    return init(this, data, byteOffset, length);\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.typed.float64-array.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.typed.int16-array.js":
/*!***************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.typed.int16-array.js ***!
  \***************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("__webpack_require__(/*! ./_typed-array */ \"./node_modules/core-js/modules/_typed-array.js\")('Int16', 2, function (init) {\r\n  return function Int16Array(data, byteOffset, length) {\r\n    return init(this, data, byteOffset, length);\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.typed.int16-array.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.typed.int32-array.js":
/*!***************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.typed.int32-array.js ***!
  \***************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("__webpack_require__(/*! ./_typed-array */ \"./node_modules/core-js/modules/_typed-array.js\")('Int32', 4, function (init) {\r\n  return function Int32Array(data, byteOffset, length) {\r\n    return init(this, data, byteOffset, length);\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.typed.int32-array.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.typed.int8-array.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.typed.int8-array.js ***!
  \**************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("__webpack_require__(/*! ./_typed-array */ \"./node_modules/core-js/modules/_typed-array.js\")('Int8', 1, function (init) {\r\n  return function Int8Array(data, byteOffset, length) {\r\n    return init(this, data, byteOffset, length);\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.typed.int8-array.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.typed.uint16-array.js":
/*!****************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.typed.uint16-array.js ***!
  \****************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("__webpack_require__(/*! ./_typed-array */ \"./node_modules/core-js/modules/_typed-array.js\")('Uint16', 2, function (init) {\r\n  return function Uint16Array(data, byteOffset, length) {\r\n    return init(this, data, byteOffset, length);\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.typed.uint16-array.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.typed.uint32-array.js":
/*!****************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.typed.uint32-array.js ***!
  \****************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("__webpack_require__(/*! ./_typed-array */ \"./node_modules/core-js/modules/_typed-array.js\")('Uint32', 4, function (init) {\r\n  return function Uint32Array(data, byteOffset, length) {\r\n    return init(this, data, byteOffset, length);\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.typed.uint32-array.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.typed.uint8-array.js":
/*!***************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.typed.uint8-array.js ***!
  \***************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("__webpack_require__(/*! ./_typed-array */ \"./node_modules/core-js/modules/_typed-array.js\")('Uint8', 1, function (init) {\r\n  return function Uint8Array(data, byteOffset, length) {\r\n    return init(this, data, byteOffset, length);\r\n  };\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.typed.uint8-array.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.typed.uint8-clamped-array.js":
/*!***********************************************************************!*\
  !*** ./node_modules/core-js/modules/es6.typed.uint8-clamped-array.js ***!
  \***********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("__webpack_require__(/*! ./_typed-array */ \"./node_modules/core-js/modules/_typed-array.js\")('Uint8', 1, function (init) {\r\n  return function Uint8ClampedArray(data, byteOffset, length) {\r\n    return init(this, data, byteOffset, length);\r\n  };\r\n}, true);\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.typed.uint8-clamped-array.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.weak-map.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/es6.weak-map.js ***!
  \******************************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar each = __webpack_require__(/*! ./_array-methods */ \"./node_modules/core-js/modules/_array-methods.js\")(0);\r\nvar redefine = __webpack_require__(/*! ./_redefine */ \"./node_modules/core-js/modules/_redefine.js\");\r\nvar meta = __webpack_require__(/*! ./_meta */ \"./node_modules/core-js/modules/_meta.js\");\r\nvar assign = __webpack_require__(/*! ./_object-assign */ \"./node_modules/core-js/modules/_object-assign.js\");\r\nvar weak = __webpack_require__(/*! ./_collection-weak */ \"./node_modules/core-js/modules/_collection-weak.js\");\r\nvar isObject = __webpack_require__(/*! ./_is-object */ \"./node_modules/core-js/modules/_is-object.js\");\r\nvar validate = __webpack_require__(/*! ./_validate-collection */ \"./node_modules/core-js/modules/_validate-collection.js\");\r\nvar NATIVE_WEAK_MAP = __webpack_require__(/*! ./_validate-collection */ \"./node_modules/core-js/modules/_validate-collection.js\");\r\nvar IS_IE11 = !global.ActiveXObject && 'ActiveXObject' in global;\r\nvar WEAK_MAP = 'WeakMap';\r\nvar getWeak = meta.getWeak;\r\nvar isExtensible = Object.isExtensible;\r\nvar uncaughtFrozenStore = weak.ufstore;\r\nvar InternalMap;\r\n\r\nvar wrapper = function (get) {\r\n  return function WeakMap() {\r\n    return get(this, arguments.length > 0 ? arguments[0] : undefined);\r\n  };\r\n};\r\n\r\nvar methods = {\r\n  // 23.3.3.3 WeakMap.prototype.get(key)\r\n  get: function get(key) {\r\n    if (isObject(key)) {\r\n      var data = getWeak(key);\r\n      if (data === true) return uncaughtFrozenStore(validate(this, WEAK_MAP)).get(key);\r\n      return data ? data[this._i] : undefined;\r\n    }\r\n  },\r\n  // 23.3.3.5 WeakMap.prototype.set(key, value)\r\n  set: function set(key, value) {\r\n    return weak.def(validate(this, WEAK_MAP), key, value);\r\n  }\r\n};\r\n\r\n// 23.3 WeakMap Objects\r\nvar $WeakMap = module.exports = __webpack_require__(/*! ./_collection */ \"./node_modules/core-js/modules/_collection.js\")(WEAK_MAP, wrapper, methods, weak, true, true);\r\n\r\n// IE11 WeakMap frozen keys fix\r\nif (NATIVE_WEAK_MAP && IS_IE11) {\r\n  InternalMap = weak.getConstructor(wrapper, WEAK_MAP);\r\n  assign(InternalMap.prototype, methods);\r\n  meta.NEED = true;\r\n  each(['delete', 'has', 'get', 'set'], function (key) {\r\n    var proto = $WeakMap.prototype;\r\n    var method = proto[key];\r\n    redefine(proto, key, function (a, b) {\r\n      // store frozen objects on internal weakmap shim\r\n      if (isObject(a) && !isExtensible(a)) {\r\n        if (!this._f) this._f = new InternalMap();\r\n        var result = this._f[key](a, b);\r\n        return key == 'set' ? this : result;\r\n      // store all the rest on native weakmap\r\n      } return method.call(this, a, b);\r\n    });\r\n  });\r\n}\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.weak-map.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es6.weak-set.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/es6.weak-set.js ***!
  \******************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar weak = __webpack_require__(/*! ./_collection-weak */ \"./node_modules/core-js/modules/_collection-weak.js\");\r\nvar validate = __webpack_require__(/*! ./_validate-collection */ \"./node_modules/core-js/modules/_validate-collection.js\");\r\nvar WEAK_SET = 'WeakSet';\r\n\r\n// 23.4 WeakSet Objects\r\n__webpack_require__(/*! ./_collection */ \"./node_modules/core-js/modules/_collection.js\")(WEAK_SET, function (get) {\r\n  return function WeakSet() { return get(this, arguments.length > 0 ? arguments[0] : undefined); };\r\n}, {\r\n  // 23.4.3.1 WeakSet.prototype.add(value)\r\n  add: function add(value) {\r\n    return weak.def(validate(this, WEAK_SET), value, true);\r\n  }\r\n}, weak, false, true);\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es6.weak-set.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.array.flat-map.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.array.flat-map.js ***!
  \************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// https://tc39.github.io/proposal-flatMap/#sec-Array.prototype.flatMap\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar flattenIntoArray = __webpack_require__(/*! ./_flatten-into-array */ \"./node_modules/core-js/modules/_flatten-into-array.js\");\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nvar aFunction = __webpack_require__(/*! ./_a-function */ \"./node_modules/core-js/modules/_a-function.js\");\r\nvar arraySpeciesCreate = __webpack_require__(/*! ./_array-species-create */ \"./node_modules/core-js/modules/_array-species-create.js\");\r\n\r\n$export($export.P, 'Array', {\r\n  flatMap: function flatMap(callbackfn /* , thisArg */) {\r\n    var O = toObject(this);\r\n    var sourceLen, A;\r\n    aFunction(callbackfn);\r\n    sourceLen = toLength(O.length);\r\n    A = arraySpeciesCreate(O, 0);\r\n    flattenIntoArray(A, O, O, sourceLen, 0, 1, callbackfn, arguments[1]);\r\n    return A;\r\n  }\r\n});\r\n\r\n__webpack_require__(/*! ./_add-to-unscopables */ \"./node_modules/core-js/modules/_add-to-unscopables.js\")('flatMap');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.array.flat-map.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.array.flatten.js":
/*!***********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.array.flatten.js ***!
  \***********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// https://tc39.github.io/proposal-flatMap/#sec-Array.prototype.flatten\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar flattenIntoArray = __webpack_require__(/*! ./_flatten-into-array */ \"./node_modules/core-js/modules/_flatten-into-array.js\");\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nvar toInteger = __webpack_require__(/*! ./_to-integer */ \"./node_modules/core-js/modules/_to-integer.js\");\r\nvar arraySpeciesCreate = __webpack_require__(/*! ./_array-species-create */ \"./node_modules/core-js/modules/_array-species-create.js\");\r\n\r\n$export($export.P, 'Array', {\r\n  flatten: function flatten(/* depthArg = 1 */) {\r\n    var depthArg = arguments[0];\r\n    var O = toObject(this);\r\n    var sourceLen = toLength(O.length);\r\n    var A = arraySpeciesCreate(O, 0);\r\n    flattenIntoArray(A, O, O, sourceLen, 0, depthArg === undefined ? 1 : toInteger(depthArg));\r\n    return A;\r\n  }\r\n});\r\n\r\n__webpack_require__(/*! ./_add-to-unscopables */ \"./node_modules/core-js/modules/_add-to-unscopables.js\")('flatten');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.array.flatten.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.array.includes.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.array.includes.js ***!
  \************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// https://github.com/tc39/Array.prototype.includes\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $includes = __webpack_require__(/*! ./_array-includes */ \"./node_modules/core-js/modules/_array-includes.js\")(true);\r\n\r\n$export($export.P, 'Array', {\r\n  includes: function includes(el /* , fromIndex = 0 */) {\r\n    return $includes(this, el, arguments.length > 1 ? arguments[1] : undefined);\r\n  }\r\n});\r\n\r\n__webpack_require__(/*! ./_add-to-unscopables */ \"./node_modules/core-js/modules/_add-to-unscopables.js\")('includes');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.array.includes.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.asap.js":
/*!**************************************************!*\
  !*** ./node_modules/core-js/modules/es7.asap.js ***!
  \**************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://github.com/rwaldron/tc39-notes/blob/master/es6/2014-09/sept-25.md#510-globalasap-for-enqueuing-a-microtask\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar microtask = __webpack_require__(/*! ./_microtask */ \"./node_modules/core-js/modules/_microtask.js\")();\r\nvar process = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\").process;\r\nvar isNode = __webpack_require__(/*! ./_cof */ \"./node_modules/core-js/modules/_cof.js\")(process) == 'process';\r\n\r\n$export($export.G, {\r\n  asap: function asap(fn) {\r\n    var domain = isNode && process.domain;\r\n    microtask(domain ? domain.bind(fn) : fn);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.asap.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.error.is-error.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.error.is-error.js ***!
  \************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://github.com/ljharb/proposal-is-error\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar cof = __webpack_require__(/*! ./_cof */ \"./node_modules/core-js/modules/_cof.js\");\r\n\r\n$export($export.S, 'Error', {\r\n  isError: function isError(it) {\r\n    return cof(it) === 'Error';\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.error.is-error.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.global.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/es7.global.js ***!
  \****************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://github.com/tc39/proposal-global\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.G, { global: __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\") });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.global.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.map.from.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/es7.map.from.js ***!
  \******************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://tc39.github.io/proposal-setmap-offrom/#sec-map.from\r\n__webpack_require__(/*! ./_set-collection-from */ \"./node_modules/core-js/modules/_set-collection-from.js\")('Map');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.map.from.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.map.of.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/es7.map.of.js ***!
  \****************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://tc39.github.io/proposal-setmap-offrom/#sec-map.of\r\n__webpack_require__(/*! ./_set-collection-of */ \"./node_modules/core-js/modules/_set-collection-of.js\")('Map');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.map.of.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.map.to-json.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.map.to-json.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://github.com/DavidBruant/Map-Set.prototype.toJSON\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.P + $export.R, 'Map', { toJSON: __webpack_require__(/*! ./_collection-to-json */ \"./node_modules/core-js/modules/_collection-to-json.js\")('Map') });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.map.to-json.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.math.clamp.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.math.clamp.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://rwaldron.github.io/proposal-math-extensions/\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Math', {\r\n  clamp: function clamp(x, lower, upper) {\r\n    return Math.min(upper, Math.max(lower, x));\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.math.clamp.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.math.deg-per-rad.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.math.deg-per-rad.js ***!
  \**************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://rwaldron.github.io/proposal-math-extensions/\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Math', { DEG_PER_RAD: Math.PI / 180 });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.math.deg-per-rad.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.math.degrees.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.math.degrees.js ***!
  \**********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://rwaldron.github.io/proposal-math-extensions/\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar RAD_PER_DEG = 180 / Math.PI;\r\n\r\n$export($export.S, 'Math', {\r\n  degrees: function degrees(radians) {\r\n    return radians * RAD_PER_DEG;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.math.degrees.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.math.fscale.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.math.fscale.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://rwaldron.github.io/proposal-math-extensions/\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar scale = __webpack_require__(/*! ./_math-scale */ \"./node_modules/core-js/modules/_math-scale.js\");\r\nvar fround = __webpack_require__(/*! ./_math-fround */ \"./node_modules/core-js/modules/_math-fround.js\");\r\n\r\n$export($export.S, 'Math', {\r\n  fscale: function fscale(x, inLow, inHigh, outLow, outHigh) {\r\n    return fround(scale(x, inLow, inHigh, outLow, outHigh));\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.math.fscale.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.math.iaddh.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.math.iaddh.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://gist.github.com/BrendanEich/4294d5c212a6d2254703\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Math', {\r\n  iaddh: function iaddh(x0, x1, y0, y1) {\r\n    var $x0 = x0 >>> 0;\r\n    var $x1 = x1 >>> 0;\r\n    var $y0 = y0 >>> 0;\r\n    return $x1 + (y1 >>> 0) + (($x0 & $y0 | ($x0 | $y0) & ~($x0 + $y0 >>> 0)) >>> 31) | 0;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.math.iaddh.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.math.imulh.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.math.imulh.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://gist.github.com/BrendanEich/4294d5c212a6d2254703\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Math', {\r\n  imulh: function imulh(u, v) {\r\n    var UINT16 = 0xffff;\r\n    var $u = +u;\r\n    var $v = +v;\r\n    var u0 = $u & UINT16;\r\n    var v0 = $v & UINT16;\r\n    var u1 = $u >> 16;\r\n    var v1 = $v >> 16;\r\n    var t = (u1 * v0 >>> 0) + (u0 * v0 >>> 16);\r\n    return u1 * v1 + (t >> 16) + ((u0 * v1 >>> 0) + (t & UINT16) >> 16);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.math.imulh.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.math.isubh.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.math.isubh.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://gist.github.com/BrendanEich/4294d5c212a6d2254703\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Math', {\r\n  isubh: function isubh(x0, x1, y0, y1) {\r\n    var $x0 = x0 >>> 0;\r\n    var $x1 = x1 >>> 0;\r\n    var $y0 = y0 >>> 0;\r\n    return $x1 - (y1 >>> 0) - ((~$x0 & $y0 | ~($x0 ^ $y0) & $x0 - $y0 >>> 0) >>> 31) | 0;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.math.isubh.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.math.rad-per-deg.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.math.rad-per-deg.js ***!
  \**************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://rwaldron.github.io/proposal-math-extensions/\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Math', { RAD_PER_DEG: 180 / Math.PI });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.math.rad-per-deg.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.math.radians.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.math.radians.js ***!
  \**********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://rwaldron.github.io/proposal-math-extensions/\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar DEG_PER_RAD = Math.PI / 180;\r\n\r\n$export($export.S, 'Math', {\r\n  radians: function radians(degrees) {\r\n    return degrees * DEG_PER_RAD;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.math.radians.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.math.scale.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.math.scale.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://rwaldron.github.io/proposal-math-extensions/\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Math', { scale: __webpack_require__(/*! ./_math-scale */ \"./node_modules/core-js/modules/_math-scale.js\") });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.math.scale.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.math.signbit.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.math.signbit.js ***!
  \**********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// http://jfbastien.github.io/papers/Math.signbit.html\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Math', { signbit: function signbit(x) {\r\n  // eslint-disable-next-line no-self-compare\r\n  return (x = +x) != x ? x : x == 0 ? 1 / x == Infinity : x > 0;\r\n} });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.math.signbit.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.math.umulh.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.math.umulh.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://gist.github.com/BrendanEich/4294d5c212a6d2254703\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'Math', {\r\n  umulh: function umulh(u, v) {\r\n    var UINT16 = 0xffff;\r\n    var $u = +u;\r\n    var $v = +v;\r\n    var u0 = $u & UINT16;\r\n    var v0 = $v & UINT16;\r\n    var u1 = $u >>> 16;\r\n    var v1 = $v >>> 16;\r\n    var t = (u1 * v0 >>> 0) + (u0 * v0 >>> 16);\r\n    return u1 * v1 + (t >>> 16) + ((u0 * v1 >>> 0) + (t & UINT16) >>> 16);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.math.umulh.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.object.define-getter.js":
/*!******************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.object.define-getter.js ***!
  \******************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar aFunction = __webpack_require__(/*! ./_a-function */ \"./node_modules/core-js/modules/_a-function.js\");\r\nvar $defineProperty = __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\");\r\n\r\n// B.2.2.2 Object.prototype.__defineGetter__(P, getter)\r\n__webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\") && $export($export.P + __webpack_require__(/*! ./_object-forced-pam */ \"./node_modules/core-js/modules/_object-forced-pam.js\"), 'Object', {\r\n  __defineGetter__: function __defineGetter__(P, getter) {\r\n    $defineProperty.f(toObject(this), P, { get: aFunction(getter), enumerable: true, configurable: true });\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.object.define-getter.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.object.define-setter.js":
/*!******************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.object.define-setter.js ***!
  \******************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar aFunction = __webpack_require__(/*! ./_a-function */ \"./node_modules/core-js/modules/_a-function.js\");\r\nvar $defineProperty = __webpack_require__(/*! ./_object-dp */ \"./node_modules/core-js/modules/_object-dp.js\");\r\n\r\n// B.2.2.3 Object.prototype.__defineSetter__(P, setter)\r\n__webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\") && $export($export.P + __webpack_require__(/*! ./_object-forced-pam */ \"./node_modules/core-js/modules/_object-forced-pam.js\"), 'Object', {\r\n  __defineSetter__: function __defineSetter__(P, setter) {\r\n    $defineProperty.f(toObject(this), P, { set: aFunction(setter), enumerable: true, configurable: true });\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.object.define-setter.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.object.entries.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.object.entries.js ***!
  \************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://github.com/tc39/proposal-object-values-entries\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $entries = __webpack_require__(/*! ./_object-to-array */ \"./node_modules/core-js/modules/_object-to-array.js\")(true);\r\n\r\n$export($export.S, 'Object', {\r\n  entries: function entries(it) {\r\n    return $entries(it);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.object.entries.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.object.get-own-property-descriptors.js":
/*!*********************************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.object.get-own-property-descriptors.js ***!
  \*********************************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://github.com/tc39/proposal-object-getownpropertydescriptors\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar ownKeys = __webpack_require__(/*! ./_own-keys */ \"./node_modules/core-js/modules/_own-keys.js\");\r\nvar toIObject = __webpack_require__(/*! ./_to-iobject */ \"./node_modules/core-js/modules/_to-iobject.js\");\r\nvar gOPD = __webpack_require__(/*! ./_object-gopd */ \"./node_modules/core-js/modules/_object-gopd.js\");\r\nvar createProperty = __webpack_require__(/*! ./_create-property */ \"./node_modules/core-js/modules/_create-property.js\");\r\n\r\n$export($export.S, 'Object', {\r\n  getOwnPropertyDescriptors: function getOwnPropertyDescriptors(object) {\r\n    var O = toIObject(object);\r\n    var getDesc = gOPD.f;\r\n    var keys = ownKeys(O);\r\n    var result = {};\r\n    var i = 0;\r\n    var key, desc;\r\n    while (keys.length > i) {\r\n      desc = getDesc(O, key = keys[i++]);\r\n      if (desc !== undefined) createProperty(result, key, desc);\r\n    }\r\n    return result;\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.object.get-own-property-descriptors.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.object.lookup-getter.js":
/*!******************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.object.lookup-getter.js ***!
  \******************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar toPrimitive = __webpack_require__(/*! ./_to-primitive */ \"./node_modules/core-js/modules/_to-primitive.js\");\r\nvar getPrototypeOf = __webpack_require__(/*! ./_object-gpo */ \"./node_modules/core-js/modules/_object-gpo.js\");\r\nvar getOwnPropertyDescriptor = __webpack_require__(/*! ./_object-gopd */ \"./node_modules/core-js/modules/_object-gopd.js\").f;\r\n\r\n// B.2.2.4 Object.prototype.__lookupGetter__(P)\r\n__webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\") && $export($export.P + __webpack_require__(/*! ./_object-forced-pam */ \"./node_modules/core-js/modules/_object-forced-pam.js\"), 'Object', {\r\n  __lookupGetter__: function __lookupGetter__(P) {\r\n    var O = toObject(this);\r\n    var K = toPrimitive(P, true);\r\n    var D;\r\n    do {\r\n      if (D = getOwnPropertyDescriptor(O, K)) return D.get;\r\n    } while (O = getPrototypeOf(O));\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.object.lookup-getter.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.object.lookup-setter.js":
/*!******************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.object.lookup-setter.js ***!
  \******************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar toObject = __webpack_require__(/*! ./_to-object */ \"./node_modules/core-js/modules/_to-object.js\");\r\nvar toPrimitive = __webpack_require__(/*! ./_to-primitive */ \"./node_modules/core-js/modules/_to-primitive.js\");\r\nvar getPrototypeOf = __webpack_require__(/*! ./_object-gpo */ \"./node_modules/core-js/modules/_object-gpo.js\");\r\nvar getOwnPropertyDescriptor = __webpack_require__(/*! ./_object-gopd */ \"./node_modules/core-js/modules/_object-gopd.js\").f;\r\n\r\n// B.2.2.5 Object.prototype.__lookupSetter__(P)\r\n__webpack_require__(/*! ./_descriptors */ \"./node_modules/core-js/modules/_descriptors.js\") && $export($export.P + __webpack_require__(/*! ./_object-forced-pam */ \"./node_modules/core-js/modules/_object-forced-pam.js\"), 'Object', {\r\n  __lookupSetter__: function __lookupSetter__(P) {\r\n    var O = toObject(this);\r\n    var K = toPrimitive(P, true);\r\n    var D;\r\n    do {\r\n      if (D = getOwnPropertyDescriptor(O, K)) return D.set;\r\n    } while (O = getPrototypeOf(O));\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.object.lookup-setter.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.object.values.js":
/*!***********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.object.values.js ***!
  \***********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://github.com/tc39/proposal-object-values-entries\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $values = __webpack_require__(/*! ./_object-to-array */ \"./node_modules/core-js/modules/_object-to-array.js\")(false);\r\n\r\n$export($export.S, 'Object', {\r\n  values: function values(it) {\r\n    return $values(it);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.object.values.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.observable.js":
/*!********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.observable.js ***!
  \********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// https://github.com/zenparsing/es-observable\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar core = __webpack_require__(/*! ./_core */ \"./node_modules/core-js/modules/_core.js\");\r\nvar microtask = __webpack_require__(/*! ./_microtask */ \"./node_modules/core-js/modules/_microtask.js\")();\r\nvar OBSERVABLE = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\")('observable');\r\nvar aFunction = __webpack_require__(/*! ./_a-function */ \"./node_modules/core-js/modules/_a-function.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar anInstance = __webpack_require__(/*! ./_an-instance */ \"./node_modules/core-js/modules/_an-instance.js\");\r\nvar redefineAll = __webpack_require__(/*! ./_redefine-all */ \"./node_modules/core-js/modules/_redefine-all.js\");\r\nvar hide = __webpack_require__(/*! ./_hide */ \"./node_modules/core-js/modules/_hide.js\");\r\nvar forOf = __webpack_require__(/*! ./_for-of */ \"./node_modules/core-js/modules/_for-of.js\");\r\nvar RETURN = forOf.RETURN;\r\n\r\nvar getMethod = function (fn) {\r\n  return fn == null ? undefined : aFunction(fn);\r\n};\r\n\r\nvar cleanupSubscription = function (subscription) {\r\n  var cleanup = subscription._c;\r\n  if (cleanup) {\r\n    subscription._c = undefined;\r\n    cleanup();\r\n  }\r\n};\r\n\r\nvar subscriptionClosed = function (subscription) {\r\n  return subscription._o === undefined;\r\n};\r\n\r\nvar closeSubscription = function (subscription) {\r\n  if (!subscriptionClosed(subscription)) {\r\n    subscription._o = undefined;\r\n    cleanupSubscription(subscription);\r\n  }\r\n};\r\n\r\nvar Subscription = function (observer, subscriber) {\r\n  anObject(observer);\r\n  this._c = undefined;\r\n  this._o = observer;\r\n  observer = new SubscriptionObserver(this);\r\n  try {\r\n    var cleanup = subscriber(observer);\r\n    var subscription = cleanup;\r\n    if (cleanup != null) {\r\n      if (typeof cleanup.unsubscribe === 'function') cleanup = function () { subscription.unsubscribe(); };\r\n      else aFunction(cleanup);\r\n      this._c = cleanup;\r\n    }\r\n  } catch (e) {\r\n    observer.error(e);\r\n    return;\r\n  } if (subscriptionClosed(this)) cleanupSubscription(this);\r\n};\r\n\r\nSubscription.prototype = redefineAll({}, {\r\n  unsubscribe: function unsubscribe() { closeSubscription(this); }\r\n});\r\n\r\nvar SubscriptionObserver = function (subscription) {\r\n  this._s = subscription;\r\n};\r\n\r\nSubscriptionObserver.prototype = redefineAll({}, {\r\n  next: function next(value) {\r\n    var subscription = this._s;\r\n    if (!subscriptionClosed(subscription)) {\r\n      var observer = subscription._o;\r\n      try {\r\n        var m = getMethod(observer.next);\r\n        if (m) return m.call(observer, value);\r\n      } catch (e) {\r\n        try {\r\n          closeSubscription(subscription);\r\n        } finally {\r\n          throw e;\r\n        }\r\n      }\r\n    }\r\n  },\r\n  error: function error(value) {\r\n    var subscription = this._s;\r\n    if (subscriptionClosed(subscription)) throw value;\r\n    var observer = subscription._o;\r\n    subscription._o = undefined;\r\n    try {\r\n      var m = getMethod(observer.error);\r\n      if (!m) throw value;\r\n      value = m.call(observer, value);\r\n    } catch (e) {\r\n      try {\r\n        cleanupSubscription(subscription);\r\n      } finally {\r\n        throw e;\r\n      }\r\n    } cleanupSubscription(subscription);\r\n    return value;\r\n  },\r\n  complete: function complete(value) {\r\n    var subscription = this._s;\r\n    if (!subscriptionClosed(subscription)) {\r\n      var observer = subscription._o;\r\n      subscription._o = undefined;\r\n      try {\r\n        var m = getMethod(observer.complete);\r\n        value = m ? m.call(observer, value) : undefined;\r\n      } catch (e) {\r\n        try {\r\n          cleanupSubscription(subscription);\r\n        } finally {\r\n          throw e;\r\n        }\r\n      } cleanupSubscription(subscription);\r\n      return value;\r\n    }\r\n  }\r\n});\r\n\r\nvar $Observable = function Observable(subscriber) {\r\n  anInstance(this, $Observable, 'Observable', '_f')._f = aFunction(subscriber);\r\n};\r\n\r\nredefineAll($Observable.prototype, {\r\n  subscribe: function subscribe(observer) {\r\n    return new Subscription(observer, this._f);\r\n  },\r\n  forEach: function forEach(fn) {\r\n    var that = this;\r\n    return new (core.Promise || global.Promise)(function (resolve, reject) {\r\n      aFunction(fn);\r\n      var subscription = that.subscribe({\r\n        next: function (value) {\r\n          try {\r\n            return fn(value);\r\n          } catch (e) {\r\n            reject(e);\r\n            subscription.unsubscribe();\r\n          }\r\n        },\r\n        error: reject,\r\n        complete: resolve\r\n      });\r\n    });\r\n  }\r\n});\r\n\r\nredefineAll($Observable, {\r\n  from: function from(x) {\r\n    var C = typeof this === 'function' ? this : $Observable;\r\n    var method = getMethod(anObject(x)[OBSERVABLE]);\r\n    if (method) {\r\n      var observable = anObject(method.call(x));\r\n      return observable.constructor === C ? observable : new C(function (observer) {\r\n        return observable.subscribe(observer);\r\n      });\r\n    }\r\n    return new C(function (observer) {\r\n      var done = false;\r\n      microtask(function () {\r\n        if (!done) {\r\n          try {\r\n            if (forOf(x, false, function (it) {\r\n              observer.next(it);\r\n              if (done) return RETURN;\r\n            }) === RETURN) return;\r\n          } catch (e) {\r\n            if (done) throw e;\r\n            observer.error(e);\r\n            return;\r\n          } observer.complete();\r\n        }\r\n      });\r\n      return function () { done = true; };\r\n    });\r\n  },\r\n  of: function of() {\r\n    for (var i = 0, l = arguments.length, items = new Array(l); i < l;) items[i] = arguments[i++];\r\n    return new (typeof this === 'function' ? this : $Observable)(function (observer) {\r\n      var done = false;\r\n      microtask(function () {\r\n        if (!done) {\r\n          for (var j = 0; j < items.length; ++j) {\r\n            observer.next(items[j]);\r\n            if (done) return;\r\n          } observer.complete();\r\n        }\r\n      });\r\n      return function () { done = true; };\r\n    });\r\n  }\r\n});\r\n\r\nhide($Observable.prototype, OBSERVABLE, function () { return this; });\r\n\r\n$export($export.G, { Observable: $Observable });\r\n\r\n__webpack_require__(/*! ./_set-species */ \"./node_modules/core-js/modules/_set-species.js\")('Observable');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.observable.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.promise.finally.js":
/*!*************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.promise.finally.js ***!
  \*************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("// https://github.com/tc39/proposal-promise-finally\r\n\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar core = __webpack_require__(/*! ./_core */ \"./node_modules/core-js/modules/_core.js\");\r\nvar global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar speciesConstructor = __webpack_require__(/*! ./_species-constructor */ \"./node_modules/core-js/modules/_species-constructor.js\");\r\nvar promiseResolve = __webpack_require__(/*! ./_promise-resolve */ \"./node_modules/core-js/modules/_promise-resolve.js\");\r\n\r\n$export($export.P + $export.R, 'Promise', { 'finally': function (onFinally) {\r\n  var C = speciesConstructor(this, core.Promise || global.Promise);\r\n  var isFunction = typeof onFinally == 'function';\r\n  return this.then(\r\n    isFunction ? function (x) {\r\n      return promiseResolve(C, onFinally()).then(function () { return x; });\r\n    } : onFinally,\r\n    isFunction ? function (e) {\r\n      return promiseResolve(C, onFinally()).then(function () { throw e; });\r\n    } : onFinally\r\n  );\r\n} });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.promise.finally.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.promise.try.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.promise.try.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// https://github.com/tc39/proposal-promise-try\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar newPromiseCapability = __webpack_require__(/*! ./_new-promise-capability */ \"./node_modules/core-js/modules/_new-promise-capability.js\");\r\nvar perform = __webpack_require__(/*! ./_perform */ \"./node_modules/core-js/modules/_perform.js\");\r\n\r\n$export($export.S, 'Promise', { 'try': function (callbackfn) {\r\n  var promiseCapability = newPromiseCapability.f(this);\r\n  var result = perform(callbackfn);\r\n  (result.e ? promiseCapability.reject : promiseCapability.resolve)(result.v);\r\n  return promiseCapability.promise;\r\n} });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.promise.try.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.reflect.define-metadata.js":
/*!*********************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.reflect.define-metadata.js ***!
  \*********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var metadata = __webpack_require__(/*! ./_metadata */ \"./node_modules/core-js/modules/_metadata.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar toMetaKey = metadata.key;\r\nvar ordinaryDefineOwnMetadata = metadata.set;\r\n\r\nmetadata.exp({ defineMetadata: function defineMetadata(metadataKey, metadataValue, target, targetKey) {\r\n  ordinaryDefineOwnMetadata(metadataKey, metadataValue, anObject(target), toMetaKey(targetKey));\r\n} });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.reflect.define-metadata.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.reflect.delete-metadata.js":
/*!*********************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.reflect.delete-metadata.js ***!
  \*********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var metadata = __webpack_require__(/*! ./_metadata */ \"./node_modules/core-js/modules/_metadata.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar toMetaKey = metadata.key;\r\nvar getOrCreateMetadataMap = metadata.map;\r\nvar store = metadata.store;\r\n\r\nmetadata.exp({ deleteMetadata: function deleteMetadata(metadataKey, target /* , targetKey */) {\r\n  var targetKey = arguments.length < 3 ? undefined : toMetaKey(arguments[2]);\r\n  var metadataMap = getOrCreateMetadataMap(anObject(target), targetKey, false);\r\n  if (metadataMap === undefined || !metadataMap['delete'](metadataKey)) return false;\r\n  if (metadataMap.size) return true;\r\n  var targetMetadata = store.get(target);\r\n  targetMetadata['delete'](targetKey);\r\n  return !!targetMetadata.size || store['delete'](target);\r\n} });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.reflect.delete-metadata.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.reflect.get-metadata-keys.js":
/*!***********************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.reflect.get-metadata-keys.js ***!
  \***********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var Set = __webpack_require__(/*! ./es6.set */ \"./node_modules/core-js/modules/es6.set.js\");\r\nvar from = __webpack_require__(/*! ./_array-from-iterable */ \"./node_modules/core-js/modules/_array-from-iterable.js\");\r\nvar metadata = __webpack_require__(/*! ./_metadata */ \"./node_modules/core-js/modules/_metadata.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar getPrototypeOf = __webpack_require__(/*! ./_object-gpo */ \"./node_modules/core-js/modules/_object-gpo.js\");\r\nvar ordinaryOwnMetadataKeys = metadata.keys;\r\nvar toMetaKey = metadata.key;\r\n\r\nvar ordinaryMetadataKeys = function (O, P) {\r\n  var oKeys = ordinaryOwnMetadataKeys(O, P);\r\n  var parent = getPrototypeOf(O);\r\n  if (parent === null) return oKeys;\r\n  var pKeys = ordinaryMetadataKeys(parent, P);\r\n  return pKeys.length ? oKeys.length ? from(new Set(oKeys.concat(pKeys))) : pKeys : oKeys;\r\n};\r\n\r\nmetadata.exp({ getMetadataKeys: function getMetadataKeys(target /* , targetKey */) {\r\n  return ordinaryMetadataKeys(anObject(target), arguments.length < 2 ? undefined : toMetaKey(arguments[1]));\r\n} });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.reflect.get-metadata-keys.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.reflect.get-metadata.js":
/*!******************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.reflect.get-metadata.js ***!
  \******************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var metadata = __webpack_require__(/*! ./_metadata */ \"./node_modules/core-js/modules/_metadata.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar getPrototypeOf = __webpack_require__(/*! ./_object-gpo */ \"./node_modules/core-js/modules/_object-gpo.js\");\r\nvar ordinaryHasOwnMetadata = metadata.has;\r\nvar ordinaryGetOwnMetadata = metadata.get;\r\nvar toMetaKey = metadata.key;\r\n\r\nvar ordinaryGetMetadata = function (MetadataKey, O, P) {\r\n  var hasOwn = ordinaryHasOwnMetadata(MetadataKey, O, P);\r\n  if (hasOwn) return ordinaryGetOwnMetadata(MetadataKey, O, P);\r\n  var parent = getPrototypeOf(O);\r\n  return parent !== null ? ordinaryGetMetadata(MetadataKey, parent, P) : undefined;\r\n};\r\n\r\nmetadata.exp({ getMetadata: function getMetadata(metadataKey, target /* , targetKey */) {\r\n  return ordinaryGetMetadata(metadataKey, anObject(target), arguments.length < 3 ? undefined : toMetaKey(arguments[2]));\r\n} });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.reflect.get-metadata.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.reflect.get-own-metadata-keys.js":
/*!***************************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.reflect.get-own-metadata-keys.js ***!
  \***************************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var metadata = __webpack_require__(/*! ./_metadata */ \"./node_modules/core-js/modules/_metadata.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar ordinaryOwnMetadataKeys = metadata.keys;\r\nvar toMetaKey = metadata.key;\r\n\r\nmetadata.exp({ getOwnMetadataKeys: function getOwnMetadataKeys(target /* , targetKey */) {\r\n  return ordinaryOwnMetadataKeys(anObject(target), arguments.length < 2 ? undefined : toMetaKey(arguments[1]));\r\n} });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.reflect.get-own-metadata-keys.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.reflect.get-own-metadata.js":
/*!**********************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.reflect.get-own-metadata.js ***!
  \**********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var metadata = __webpack_require__(/*! ./_metadata */ \"./node_modules/core-js/modules/_metadata.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar ordinaryGetOwnMetadata = metadata.get;\r\nvar toMetaKey = metadata.key;\r\n\r\nmetadata.exp({ getOwnMetadata: function getOwnMetadata(metadataKey, target /* , targetKey */) {\r\n  return ordinaryGetOwnMetadata(metadataKey, anObject(target)\r\n    , arguments.length < 3 ? undefined : toMetaKey(arguments[2]));\r\n} });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.reflect.get-own-metadata.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.reflect.has-metadata.js":
/*!******************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.reflect.has-metadata.js ***!
  \******************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var metadata = __webpack_require__(/*! ./_metadata */ \"./node_modules/core-js/modules/_metadata.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar getPrototypeOf = __webpack_require__(/*! ./_object-gpo */ \"./node_modules/core-js/modules/_object-gpo.js\");\r\nvar ordinaryHasOwnMetadata = metadata.has;\r\nvar toMetaKey = metadata.key;\r\n\r\nvar ordinaryHasMetadata = function (MetadataKey, O, P) {\r\n  var hasOwn = ordinaryHasOwnMetadata(MetadataKey, O, P);\r\n  if (hasOwn) return true;\r\n  var parent = getPrototypeOf(O);\r\n  return parent !== null ? ordinaryHasMetadata(MetadataKey, parent, P) : false;\r\n};\r\n\r\nmetadata.exp({ hasMetadata: function hasMetadata(metadataKey, target /* , targetKey */) {\r\n  return ordinaryHasMetadata(metadataKey, anObject(target), arguments.length < 3 ? undefined : toMetaKey(arguments[2]));\r\n} });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.reflect.has-metadata.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.reflect.has-own-metadata.js":
/*!**********************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.reflect.has-own-metadata.js ***!
  \**********************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var metadata = __webpack_require__(/*! ./_metadata */ \"./node_modules/core-js/modules/_metadata.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar ordinaryHasOwnMetadata = metadata.has;\r\nvar toMetaKey = metadata.key;\r\n\r\nmetadata.exp({ hasOwnMetadata: function hasOwnMetadata(metadataKey, target /* , targetKey */) {\r\n  return ordinaryHasOwnMetadata(metadataKey, anObject(target)\r\n    , arguments.length < 3 ? undefined : toMetaKey(arguments[2]));\r\n} });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.reflect.has-own-metadata.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.reflect.metadata.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.reflect.metadata.js ***!
  \**************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var $metadata = __webpack_require__(/*! ./_metadata */ \"./node_modules/core-js/modules/_metadata.js\");\r\nvar anObject = __webpack_require__(/*! ./_an-object */ \"./node_modules/core-js/modules/_an-object.js\");\r\nvar aFunction = __webpack_require__(/*! ./_a-function */ \"./node_modules/core-js/modules/_a-function.js\");\r\nvar toMetaKey = $metadata.key;\r\nvar ordinaryDefineOwnMetadata = $metadata.set;\r\n\r\n$metadata.exp({ metadata: function metadata(metadataKey, metadataValue) {\r\n  return function decorator(target, targetKey) {\r\n    ordinaryDefineOwnMetadata(\r\n      metadataKey, metadataValue,\r\n      (targetKey !== undefined ? anObject : aFunction)(target),\r\n      toMetaKey(targetKey)\r\n    );\r\n  };\r\n} });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.reflect.metadata.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.set.from.js":
/*!******************************************************!*\
  !*** ./node_modules/core-js/modules/es7.set.from.js ***!
  \******************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://tc39.github.io/proposal-setmap-offrom/#sec-set.from\r\n__webpack_require__(/*! ./_set-collection-from */ \"./node_modules/core-js/modules/_set-collection-from.js\")('Set');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.set.from.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.set.of.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/es7.set.of.js ***!
  \****************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://tc39.github.io/proposal-setmap-offrom/#sec-set.of\r\n__webpack_require__(/*! ./_set-collection-of */ \"./node_modules/core-js/modules/_set-collection-of.js\")('Set');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.set.of.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.set.to-json.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.set.to-json.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://github.com/DavidBruant/Map-Set.prototype.toJSON\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.P + $export.R, 'Set', { toJSON: __webpack_require__(/*! ./_collection-to-json */ \"./node_modules/core-js/modules/_collection-to-json.js\")('Set') });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.set.to-json.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.string.at.js":
/*!*******************************************************!*\
  !*** ./node_modules/core-js/modules/es7.string.at.js ***!
  \*******************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// https://github.com/mathiasbynens/String.prototype.at\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $at = __webpack_require__(/*! ./_string-at */ \"./node_modules/core-js/modules/_string-at.js\")(true);\r\nvar $fails = __webpack_require__(/*! ./_fails */ \"./node_modules/core-js/modules/_fails.js\");\r\n\r\nvar FORCED = $fails(function () {\r\n  return '𠮷'.at(0) !== '𠮷';\r\n});\r\n\r\n$export($export.P + $export.F * FORCED, 'String', {\r\n  at: function at(pos) {\r\n    return $at(this, pos);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.string.at.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.string.match-all.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.string.match-all.js ***!
  \**************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// https://tc39.github.io/String.prototype.matchAll/\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar defined = __webpack_require__(/*! ./_defined */ \"./node_modules/core-js/modules/_defined.js\");\r\nvar toLength = __webpack_require__(/*! ./_to-length */ \"./node_modules/core-js/modules/_to-length.js\");\r\nvar isRegExp = __webpack_require__(/*! ./_is-regexp */ \"./node_modules/core-js/modules/_is-regexp.js\");\r\nvar getFlags = __webpack_require__(/*! ./_flags */ \"./node_modules/core-js/modules/_flags.js\");\r\nvar RegExpProto = RegExp.prototype;\r\n\r\nvar $RegExpStringIterator = function (regexp, string) {\r\n  this._r = regexp;\r\n  this._s = string;\r\n};\r\n\r\n__webpack_require__(/*! ./_iter-create */ \"./node_modules/core-js/modules/_iter-create.js\")($RegExpStringIterator, 'RegExp String', function next() {\r\n  var match = this._r.exec(this._s);\r\n  return { value: match, done: match === null };\r\n});\r\n\r\n$export($export.P, 'String', {\r\n  matchAll: function matchAll(regexp) {\r\n    defined(this);\r\n    if (!isRegExp(regexp)) throw TypeError(regexp + ' is not a regexp!');\r\n    var S = String(this);\r\n    var flags = 'flags' in RegExpProto ? String(regexp.flags) : getFlags.call(regexp);\r\n    var rx = new RegExp(regexp.source, ~flags.indexOf('g') ? flags : 'g' + flags);\r\n    rx.lastIndex = toLength(regexp.lastIndex);\r\n    return new $RegExpStringIterator(rx, S);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.string.match-all.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.string.pad-end.js":
/*!************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.string.pad-end.js ***!
  \************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// https://github.com/tc39/proposal-string-pad-start-end\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $pad = __webpack_require__(/*! ./_string-pad */ \"./node_modules/core-js/modules/_string-pad.js\");\r\nvar userAgent = __webpack_require__(/*! ./_user-agent */ \"./node_modules/core-js/modules/_user-agent.js\");\r\n\r\n// https://github.com/zloirock/core-js/issues/280\r\nvar WEBKIT_BUG = /Version\\/10\\.\\d+(\\.\\d+)?( Mobile\\/\\w+)? Safari\\//.test(userAgent);\r\n\r\n$export($export.P + $export.F * WEBKIT_BUG, 'String', {\r\n  padEnd: function padEnd(maxLength /* , fillString = ' ' */) {\r\n    return $pad(this, maxLength, arguments.length > 1 ? arguments[1] : undefined, false);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.string.pad-end.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.string.pad-start.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.string.pad-start.js ***!
  \**************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// https://github.com/tc39/proposal-string-pad-start-end\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $pad = __webpack_require__(/*! ./_string-pad */ \"./node_modules/core-js/modules/_string-pad.js\");\r\nvar userAgent = __webpack_require__(/*! ./_user-agent */ \"./node_modules/core-js/modules/_user-agent.js\");\r\n\r\n// https://github.com/zloirock/core-js/issues/280\r\nvar WEBKIT_BUG = /Version\\/10\\.\\d+(\\.\\d+)?( Mobile\\/\\w+)? Safari\\//.test(userAgent);\r\n\r\n$export($export.P + $export.F * WEBKIT_BUG, 'String', {\r\n  padStart: function padStart(maxLength /* , fillString = ' ' */) {\r\n    return $pad(this, maxLength, arguments.length > 1 ? arguments[1] : undefined, true);\r\n  }\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.string.pad-start.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.string.trim-left.js":
/*!**************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.string.trim-left.js ***!
  \**************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// https://github.com/sebmarkbage/ecmascript-string-left-right-trim\r\n__webpack_require__(/*! ./_string-trim */ \"./node_modules/core-js/modules/_string-trim.js\")('trimLeft', function ($trim) {\r\n  return function trimLeft() {\r\n    return $trim(this, 1);\r\n  };\r\n}, 'trimStart');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.string.trim-left.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.string.trim-right.js":
/*!***************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.string.trim-right.js ***!
  \***************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

"use strict";
eval("\r\n// https://github.com/sebmarkbage/ecmascript-string-left-right-trim\r\n__webpack_require__(/*! ./_string-trim */ \"./node_modules/core-js/modules/_string-trim.js\")('trimRight', function ($trim) {\r\n  return function trimRight() {\r\n    return $trim(this, 2);\r\n  };\r\n}, 'trimEnd');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.string.trim-right.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.symbol.async-iterator.js":
/*!*******************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.symbol.async-iterator.js ***!
  \*******************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("__webpack_require__(/*! ./_wks-define */ \"./node_modules/core-js/modules/_wks-define.js\")('asyncIterator');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.symbol.async-iterator.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.symbol.observable.js":
/*!***************************************************************!*\
  !*** ./node_modules/core-js/modules/es7.symbol.observable.js ***!
  \***************************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("__webpack_require__(/*! ./_wks-define */ \"./node_modules/core-js/modules/_wks-define.js\")('observable');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.symbol.observable.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.system.global.js":
/*!***********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.system.global.js ***!
  \***********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://github.com/tc39/proposal-global\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\n\r\n$export($export.S, 'System', { global: __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\") });\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.system.global.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.weak-map.from.js":
/*!***********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.weak-map.from.js ***!
  \***********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://tc39.github.io/proposal-setmap-offrom/#sec-weakmap.from\r\n__webpack_require__(/*! ./_set-collection-from */ \"./node_modules/core-js/modules/_set-collection-from.js\")('WeakMap');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.weak-map.from.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.weak-map.of.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.weak-map.of.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://tc39.github.io/proposal-setmap-offrom/#sec-weakmap.of\r\n__webpack_require__(/*! ./_set-collection-of */ \"./node_modules/core-js/modules/_set-collection-of.js\")('WeakMap');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.weak-map.of.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.weak-set.from.js":
/*!***********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.weak-set.from.js ***!
  \***********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://tc39.github.io/proposal-setmap-offrom/#sec-weakset.from\r\n__webpack_require__(/*! ./_set-collection-from */ \"./node_modules/core-js/modules/_set-collection-from.js\")('WeakSet');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.weak-set.from.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/es7.weak-set.of.js":
/*!*********************************************************!*\
  !*** ./node_modules/core-js/modules/es7.weak-set.of.js ***!
  \*********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// https://tc39.github.io/proposal-setmap-offrom/#sec-weakset.of\r\n__webpack_require__(/*! ./_set-collection-of */ \"./node_modules/core-js/modules/_set-collection-of.js\")('WeakSet');\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/es7.weak-set.of.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/web.dom.iterable.js":
/*!**********************************************************!*\
  !*** ./node_modules/core-js/modules/web.dom.iterable.js ***!
  \**********************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var $iterators = __webpack_require__(/*! ./es6.array.iterator */ \"./node_modules/core-js/modules/es6.array.iterator.js\");\r\nvar getKeys = __webpack_require__(/*! ./_object-keys */ \"./node_modules/core-js/modules/_object-keys.js\");\r\nvar redefine = __webpack_require__(/*! ./_redefine */ \"./node_modules/core-js/modules/_redefine.js\");\r\nvar global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar hide = __webpack_require__(/*! ./_hide */ \"./node_modules/core-js/modules/_hide.js\");\r\nvar Iterators = __webpack_require__(/*! ./_iterators */ \"./node_modules/core-js/modules/_iterators.js\");\r\nvar wks = __webpack_require__(/*! ./_wks */ \"./node_modules/core-js/modules/_wks.js\");\r\nvar ITERATOR = wks('iterator');\r\nvar TO_STRING_TAG = wks('toStringTag');\r\nvar ArrayValues = Iterators.Array;\r\n\r\nvar DOMIterables = {\r\n  CSSRuleList: true, // TODO: Not spec compliant, should be false.\r\n  CSSStyleDeclaration: false,\r\n  CSSValueList: false,\r\n  ClientRectList: false,\r\n  DOMRectList: false,\r\n  DOMStringList: false,\r\n  DOMTokenList: true,\r\n  DataTransferItemList: false,\r\n  FileList: false,\r\n  HTMLAllCollection: false,\r\n  HTMLCollection: false,\r\n  HTMLFormElement: false,\r\n  HTMLSelectElement: false,\r\n  MediaList: true, // TODO: Not spec compliant, should be false.\r\n  MimeTypeArray: false,\r\n  NamedNodeMap: false,\r\n  NodeList: true,\r\n  PaintRequestList: false,\r\n  Plugin: false,\r\n  PluginArray: false,\r\n  SVGLengthList: false,\r\n  SVGNumberList: false,\r\n  SVGPathSegList: false,\r\n  SVGPointList: false,\r\n  SVGStringList: false,\r\n  SVGTransformList: false,\r\n  SourceBufferList: false,\r\n  StyleSheetList: true, // TODO: Not spec compliant, should be false.\r\n  TextTrackCueList: false,\r\n  TextTrackList: false,\r\n  TouchList: false\r\n};\r\n\r\nfor (var collections = getKeys(DOMIterables), i = 0; i < collections.length; i++) {\r\n  var NAME = collections[i];\r\n  var explicit = DOMIterables[NAME];\r\n  var Collection = global[NAME];\r\n  var proto = Collection && Collection.prototype;\r\n  var key;\r\n  if (proto) {\r\n    if (!proto[ITERATOR]) hide(proto, ITERATOR, ArrayValues);\r\n    if (!proto[TO_STRING_TAG]) hide(proto, TO_STRING_TAG, NAME);\r\n    Iterators[NAME] = ArrayValues;\r\n    if (explicit) for (key in $iterators) if (!proto[key]) redefine(proto, key, $iterators[key], true);\r\n  }\r\n}\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/web.dom.iterable.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/web.immediate.js":
/*!*******************************************************!*\
  !*** ./node_modules/core-js/modules/web.immediate.js ***!
  \*******************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("var $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar $task = __webpack_require__(/*! ./_task */ \"./node_modules/core-js/modules/_task.js\");\r\n$export($export.G + $export.B, {\r\n  setImmediate: $task.set,\r\n  clearImmediate: $task.clear\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/web.immediate.js?");

/***/ }),

/***/ "./node_modules/core-js/modules/web.timers.js":
/*!****************************************************!*\
  !*** ./node_modules/core-js/modules/web.timers.js ***!
  \****************************************************/
/***/ (function(__unused_webpack_module, __unused_webpack_exports, __webpack_require__) {

eval("// ie9- setTimeout & setInterval additional parameters fix\r\nvar global = __webpack_require__(/*! ./_global */ \"./node_modules/core-js/modules/_global.js\");\r\nvar $export = __webpack_require__(/*! ./_export */ \"./node_modules/core-js/modules/_export.js\");\r\nvar userAgent = __webpack_require__(/*! ./_user-agent */ \"./node_modules/core-js/modules/_user-agent.js\");\r\nvar slice = [].slice;\r\nvar MSIE = /MSIE .\\./.test(userAgent); // <- dirty ie9- check\r\nvar wrap = function (set) {\r\n  return function (fn, time /* , ...args */) {\r\n    var boundArgs = arguments.length > 2;\r\n    var args = boundArgs ? slice.call(arguments, 2) : false;\r\n    return set(boundArgs ? function () {\r\n      // eslint-disable-next-line no-new-func\r\n      (typeof fn == 'function' ? fn : Function(fn)).apply(this, args);\r\n    } : fn, time);\r\n  };\r\n};\r\n$export($export.G + $export.B + $export.F * MSIE, {\r\n  setTimeout: wrap(global.setTimeout),\r\n  setInterval: wrap(global.setInterval)\r\n});\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/modules/web.timers.js?");

/***/ }),

/***/ "./node_modules/core-js/shim.js":
/*!**************************************!*\
  !*** ./node_modules/core-js/shim.js ***!
  \**************************************/
/***/ (function(module, __unused_webpack_exports, __webpack_require__) {

eval("__webpack_require__(/*! ./modules/es6.symbol */ \"./node_modules/core-js/modules/es6.symbol.js\");\r\n__webpack_require__(/*! ./modules/es6.object.create */ \"./node_modules/core-js/modules/es6.object.create.js\");\r\n__webpack_require__(/*! ./modules/es6.object.define-property */ \"./node_modules/core-js/modules/es6.object.define-property.js\");\r\n__webpack_require__(/*! ./modules/es6.object.define-properties */ \"./node_modules/core-js/modules/es6.object.define-properties.js\");\r\n__webpack_require__(/*! ./modules/es6.object.get-own-property-descriptor */ \"./node_modules/core-js/modules/es6.object.get-own-property-descriptor.js\");\r\n__webpack_require__(/*! ./modules/es6.object.get-prototype-of */ \"./node_modules/core-js/modules/es6.object.get-prototype-of.js\");\r\n__webpack_require__(/*! ./modules/es6.object.keys */ \"./node_modules/core-js/modules/es6.object.keys.js\");\r\n__webpack_require__(/*! ./modules/es6.object.get-own-property-names */ \"./node_modules/core-js/modules/es6.object.get-own-property-names.js\");\r\n__webpack_require__(/*! ./modules/es6.object.freeze */ \"./node_modules/core-js/modules/es6.object.freeze.js\");\r\n__webpack_require__(/*! ./modules/es6.object.seal */ \"./node_modules/core-js/modules/es6.object.seal.js\");\r\n__webpack_require__(/*! ./modules/es6.object.prevent-extensions */ \"./node_modules/core-js/modules/es6.object.prevent-extensions.js\");\r\n__webpack_require__(/*! ./modules/es6.object.is-frozen */ \"./node_modules/core-js/modules/es6.object.is-frozen.js\");\r\n__webpack_require__(/*! ./modules/es6.object.is-sealed */ \"./node_modules/core-js/modules/es6.object.is-sealed.js\");\r\n__webpack_require__(/*! ./modules/es6.object.is-extensible */ \"./node_modules/core-js/modules/es6.object.is-extensible.js\");\r\n__webpack_require__(/*! ./modules/es6.object.assign */ \"./node_modules/core-js/modules/es6.object.assign.js\");\r\n__webpack_require__(/*! ./modules/es6.object.is */ \"./node_modules/core-js/modules/es6.object.is.js\");\r\n__webpack_require__(/*! ./modules/es6.object.set-prototype-of */ \"./node_modules/core-js/modules/es6.object.set-prototype-of.js\");\r\n__webpack_require__(/*! ./modules/es6.object.to-string */ \"./node_modules/core-js/modules/es6.object.to-string.js\");\r\n__webpack_require__(/*! ./modules/es6.function.bind */ \"./node_modules/core-js/modules/es6.function.bind.js\");\r\n__webpack_require__(/*! ./modules/es6.function.name */ \"./node_modules/core-js/modules/es6.function.name.js\");\r\n__webpack_require__(/*! ./modules/es6.function.has-instance */ \"./node_modules/core-js/modules/es6.function.has-instance.js\");\r\n__webpack_require__(/*! ./modules/es6.parse-int */ \"./node_modules/core-js/modules/es6.parse-int.js\");\r\n__webpack_require__(/*! ./modules/es6.parse-float */ \"./node_modules/core-js/modules/es6.parse-float.js\");\r\n__webpack_require__(/*! ./modules/es6.number.constructor */ \"./node_modules/core-js/modules/es6.number.constructor.js\");\r\n__webpack_require__(/*! ./modules/es6.number.to-fixed */ \"./node_modules/core-js/modules/es6.number.to-fixed.js\");\r\n__webpack_require__(/*! ./modules/es6.number.to-precision */ \"./node_modules/core-js/modules/es6.number.to-precision.js\");\r\n__webpack_require__(/*! ./modules/es6.number.epsilon */ \"./node_modules/core-js/modules/es6.number.epsilon.js\");\r\n__webpack_require__(/*! ./modules/es6.number.is-finite */ \"./node_modules/core-js/modules/es6.number.is-finite.js\");\r\n__webpack_require__(/*! ./modules/es6.number.is-integer */ \"./node_modules/core-js/modules/es6.number.is-integer.js\");\r\n__webpack_require__(/*! ./modules/es6.number.is-nan */ \"./node_modules/core-js/modules/es6.number.is-nan.js\");\r\n__webpack_require__(/*! ./modules/es6.number.is-safe-integer */ \"./node_modules/core-js/modules/es6.number.is-safe-integer.js\");\r\n__webpack_require__(/*! ./modules/es6.number.max-safe-integer */ \"./node_modules/core-js/modules/es6.number.max-safe-integer.js\");\r\n__webpack_require__(/*! ./modules/es6.number.min-safe-integer */ \"./node_modules/core-js/modules/es6.number.min-safe-integer.js\");\r\n__webpack_require__(/*! ./modules/es6.number.parse-float */ \"./node_modules/core-js/modules/es6.number.parse-float.js\");\r\n__webpack_require__(/*! ./modules/es6.number.parse-int */ \"./node_modules/core-js/modules/es6.number.parse-int.js\");\r\n__webpack_require__(/*! ./modules/es6.math.acosh */ \"./node_modules/core-js/modules/es6.math.acosh.js\");\r\n__webpack_require__(/*! ./modules/es6.math.asinh */ \"./node_modules/core-js/modules/es6.math.asinh.js\");\r\n__webpack_require__(/*! ./modules/es6.math.atanh */ \"./node_modules/core-js/modules/es6.math.atanh.js\");\r\n__webpack_require__(/*! ./modules/es6.math.cbrt */ \"./node_modules/core-js/modules/es6.math.cbrt.js\");\r\n__webpack_require__(/*! ./modules/es6.math.clz32 */ \"./node_modules/core-js/modules/es6.math.clz32.js\");\r\n__webpack_require__(/*! ./modules/es6.math.cosh */ \"./node_modules/core-js/modules/es6.math.cosh.js\");\r\n__webpack_require__(/*! ./modules/es6.math.expm1 */ \"./node_modules/core-js/modules/es6.math.expm1.js\");\r\n__webpack_require__(/*! ./modules/es6.math.fround */ \"./node_modules/core-js/modules/es6.math.fround.js\");\r\n__webpack_require__(/*! ./modules/es6.math.hypot */ \"./node_modules/core-js/modules/es6.math.hypot.js\");\r\n__webpack_require__(/*! ./modules/es6.math.imul */ \"./node_modules/core-js/modules/es6.math.imul.js\");\r\n__webpack_require__(/*! ./modules/es6.math.log10 */ \"./node_modules/core-js/modules/es6.math.log10.js\");\r\n__webpack_require__(/*! ./modules/es6.math.log1p */ \"./node_modules/core-js/modules/es6.math.log1p.js\");\r\n__webpack_require__(/*! ./modules/es6.math.log2 */ \"./node_modules/core-js/modules/es6.math.log2.js\");\r\n__webpack_require__(/*! ./modules/es6.math.sign */ \"./node_modules/core-js/modules/es6.math.sign.js\");\r\n__webpack_require__(/*! ./modules/es6.math.sinh */ \"./node_modules/core-js/modules/es6.math.sinh.js\");\r\n__webpack_require__(/*! ./modules/es6.math.tanh */ \"./node_modules/core-js/modules/es6.math.tanh.js\");\r\n__webpack_require__(/*! ./modules/es6.math.trunc */ \"./node_modules/core-js/modules/es6.math.trunc.js\");\r\n__webpack_require__(/*! ./modules/es6.string.from-code-point */ \"./node_modules/core-js/modules/es6.string.from-code-point.js\");\r\n__webpack_require__(/*! ./modules/es6.string.raw */ \"./node_modules/core-js/modules/es6.string.raw.js\");\r\n__webpack_require__(/*! ./modules/es6.string.trim */ \"./node_modules/core-js/modules/es6.string.trim.js\");\r\n__webpack_require__(/*! ./modules/es6.string.iterator */ \"./node_modules/core-js/modules/es6.string.iterator.js\");\r\n__webpack_require__(/*! ./modules/es6.string.code-point-at */ \"./node_modules/core-js/modules/es6.string.code-point-at.js\");\r\n__webpack_require__(/*! ./modules/es6.string.ends-with */ \"./node_modules/core-js/modules/es6.string.ends-with.js\");\r\n__webpack_require__(/*! ./modules/es6.string.includes */ \"./node_modules/core-js/modules/es6.string.includes.js\");\r\n__webpack_require__(/*! ./modules/es6.string.repeat */ \"./node_modules/core-js/modules/es6.string.repeat.js\");\r\n__webpack_require__(/*! ./modules/es6.string.starts-with */ \"./node_modules/core-js/modules/es6.string.starts-with.js\");\r\n__webpack_require__(/*! ./modules/es6.string.anchor */ \"./node_modules/core-js/modules/es6.string.anchor.js\");\r\n__webpack_require__(/*! ./modules/es6.string.big */ \"./node_modules/core-js/modules/es6.string.big.js\");\r\n__webpack_require__(/*! ./modules/es6.string.blink */ \"./node_modules/core-js/modules/es6.string.blink.js\");\r\n__webpack_require__(/*! ./modules/es6.string.bold */ \"./node_modules/core-js/modules/es6.string.bold.js\");\r\n__webpack_require__(/*! ./modules/es6.string.fixed */ \"./node_modules/core-js/modules/es6.string.fixed.js\");\r\n__webpack_require__(/*! ./modules/es6.string.fontcolor */ \"./node_modules/core-js/modules/es6.string.fontcolor.js\");\r\n__webpack_require__(/*! ./modules/es6.string.fontsize */ \"./node_modules/core-js/modules/es6.string.fontsize.js\");\r\n__webpack_require__(/*! ./modules/es6.string.italics */ \"./node_modules/core-js/modules/es6.string.italics.js\");\r\n__webpack_require__(/*! ./modules/es6.string.link */ \"./node_modules/core-js/modules/es6.string.link.js\");\r\n__webpack_require__(/*! ./modules/es6.string.small */ \"./node_modules/core-js/modules/es6.string.small.js\");\r\n__webpack_require__(/*! ./modules/es6.string.strike */ \"./node_modules/core-js/modules/es6.string.strike.js\");\r\n__webpack_require__(/*! ./modules/es6.string.sub */ \"./node_modules/core-js/modules/es6.string.sub.js\");\r\n__webpack_require__(/*! ./modules/es6.string.sup */ \"./node_modules/core-js/modules/es6.string.sup.js\");\r\n__webpack_require__(/*! ./modules/es6.date.now */ \"./node_modules/core-js/modules/es6.date.now.js\");\r\n__webpack_require__(/*! ./modules/es6.date.to-json */ \"./node_modules/core-js/modules/es6.date.to-json.js\");\r\n__webpack_require__(/*! ./modules/es6.date.to-iso-string */ \"./node_modules/core-js/modules/es6.date.to-iso-string.js\");\r\n__webpack_require__(/*! ./modules/es6.date.to-string */ \"./node_modules/core-js/modules/es6.date.to-string.js\");\r\n__webpack_require__(/*! ./modules/es6.date.to-primitive */ \"./node_modules/core-js/modules/es6.date.to-primitive.js\");\r\n__webpack_require__(/*! ./modules/es6.array.is-array */ \"./node_modules/core-js/modules/es6.array.is-array.js\");\r\n__webpack_require__(/*! ./modules/es6.array.from */ \"./node_modules/core-js/modules/es6.array.from.js\");\r\n__webpack_require__(/*! ./modules/es6.array.of */ \"./node_modules/core-js/modules/es6.array.of.js\");\r\n__webpack_require__(/*! ./modules/es6.array.join */ \"./node_modules/core-js/modules/es6.array.join.js\");\r\n__webpack_require__(/*! ./modules/es6.array.slice */ \"./node_modules/core-js/modules/es6.array.slice.js\");\r\n__webpack_require__(/*! ./modules/es6.array.sort */ \"./node_modules/core-js/modules/es6.array.sort.js\");\r\n__webpack_require__(/*! ./modules/es6.array.for-each */ \"./node_modules/core-js/modules/es6.array.for-each.js\");\r\n__webpack_require__(/*! ./modules/es6.array.map */ \"./node_modules/core-js/modules/es6.array.map.js\");\r\n__webpack_require__(/*! ./modules/es6.array.filter */ \"./node_modules/core-js/modules/es6.array.filter.js\");\r\n__webpack_require__(/*! ./modules/es6.array.some */ \"./node_modules/core-js/modules/es6.array.some.js\");\r\n__webpack_require__(/*! ./modules/es6.array.every */ \"./node_modules/core-js/modules/es6.array.every.js\");\r\n__webpack_require__(/*! ./modules/es6.array.reduce */ \"./node_modules/core-js/modules/es6.array.reduce.js\");\r\n__webpack_require__(/*! ./modules/es6.array.reduce-right */ \"./node_modules/core-js/modules/es6.array.reduce-right.js\");\r\n__webpack_require__(/*! ./modules/es6.array.index-of */ \"./node_modules/core-js/modules/es6.array.index-of.js\");\r\n__webpack_require__(/*! ./modules/es6.array.last-index-of */ \"./node_modules/core-js/modules/es6.array.last-index-of.js\");\r\n__webpack_require__(/*! ./modules/es6.array.copy-within */ \"./node_modules/core-js/modules/es6.array.copy-within.js\");\r\n__webpack_require__(/*! ./modules/es6.array.fill */ \"./node_modules/core-js/modules/es6.array.fill.js\");\r\n__webpack_require__(/*! ./modules/es6.array.find */ \"./node_modules/core-js/modules/es6.array.find.js\");\r\n__webpack_require__(/*! ./modules/es6.array.find-index */ \"./node_modules/core-js/modules/es6.array.find-index.js\");\r\n__webpack_require__(/*! ./modules/es6.array.species */ \"./node_modules/core-js/modules/es6.array.species.js\");\r\n__webpack_require__(/*! ./modules/es6.array.iterator */ \"./node_modules/core-js/modules/es6.array.iterator.js\");\r\n__webpack_require__(/*! ./modules/es6.regexp.constructor */ \"./node_modules/core-js/modules/es6.regexp.constructor.js\");\r\n__webpack_require__(/*! ./modules/es6.regexp.exec */ \"./node_modules/core-js/modules/es6.regexp.exec.js\");\r\n__webpack_require__(/*! ./modules/es6.regexp.to-string */ \"./node_modules/core-js/modules/es6.regexp.to-string.js\");\r\n__webpack_require__(/*! ./modules/es6.regexp.flags */ \"./node_modules/core-js/modules/es6.regexp.flags.js\");\r\n__webpack_require__(/*! ./modules/es6.regexp.match */ \"./node_modules/core-js/modules/es6.regexp.match.js\");\r\n__webpack_require__(/*! ./modules/es6.regexp.replace */ \"./node_modules/core-js/modules/es6.regexp.replace.js\");\r\n__webpack_require__(/*! ./modules/es6.regexp.search */ \"./node_modules/core-js/modules/es6.regexp.search.js\");\r\n__webpack_require__(/*! ./modules/es6.regexp.split */ \"./node_modules/core-js/modules/es6.regexp.split.js\");\r\n__webpack_require__(/*! ./modules/es6.promise */ \"./node_modules/core-js/modules/es6.promise.js\");\r\n__webpack_require__(/*! ./modules/es6.map */ \"./node_modules/core-js/modules/es6.map.js\");\r\n__webpack_require__(/*! ./modules/es6.set */ \"./node_modules/core-js/modules/es6.set.js\");\r\n__webpack_require__(/*! ./modules/es6.weak-map */ \"./node_modules/core-js/modules/es6.weak-map.js\");\r\n__webpack_require__(/*! ./modules/es6.weak-set */ \"./node_modules/core-js/modules/es6.weak-set.js\");\r\n__webpack_require__(/*! ./modules/es6.typed.array-buffer */ \"./node_modules/core-js/modules/es6.typed.array-buffer.js\");\r\n__webpack_require__(/*! ./modules/es6.typed.data-view */ \"./node_modules/core-js/modules/es6.typed.data-view.js\");\r\n__webpack_require__(/*! ./modules/es6.typed.int8-array */ \"./node_modules/core-js/modules/es6.typed.int8-array.js\");\r\n__webpack_require__(/*! ./modules/es6.typed.uint8-array */ \"./node_modules/core-js/modules/es6.typed.uint8-array.js\");\r\n__webpack_require__(/*! ./modules/es6.typed.uint8-clamped-array */ \"./node_modules/core-js/modules/es6.typed.uint8-clamped-array.js\");\r\n__webpack_require__(/*! ./modules/es6.typed.int16-array */ \"./node_modules/core-js/modules/es6.typed.int16-array.js\");\r\n__webpack_require__(/*! ./modules/es6.typed.uint16-array */ \"./node_modules/core-js/modules/es6.typed.uint16-array.js\");\r\n__webpack_require__(/*! ./modules/es6.typed.int32-array */ \"./node_modules/core-js/modules/es6.typed.int32-array.js\");\r\n__webpack_require__(/*! ./modules/es6.typed.uint32-array */ \"./node_modules/core-js/modules/es6.typed.uint32-array.js\");\r\n__webpack_require__(/*! ./modules/es6.typed.float32-array */ \"./node_modules/core-js/modules/es6.typed.float32-array.js\");\r\n__webpack_require__(/*! ./modules/es6.typed.float64-array */ \"./node_modules/core-js/modules/es6.typed.float64-array.js\");\r\n__webpack_require__(/*! ./modules/es6.reflect.apply */ \"./node_modules/core-js/modules/es6.reflect.apply.js\");\r\n__webpack_require__(/*! ./modules/es6.reflect.construct */ \"./node_modules/core-js/modules/es6.reflect.construct.js\");\r\n__webpack_require__(/*! ./modules/es6.reflect.define-property */ \"./node_modules/core-js/modules/es6.reflect.define-property.js\");\r\n__webpack_require__(/*! ./modules/es6.reflect.delete-property */ \"./node_modules/core-js/modules/es6.reflect.delete-property.js\");\r\n__webpack_require__(/*! ./modules/es6.reflect.enumerate */ \"./node_modules/core-js/modules/es6.reflect.enumerate.js\");\r\n__webpack_require__(/*! ./modules/es6.reflect.get */ \"./node_modules/core-js/modules/es6.reflect.get.js\");\r\n__webpack_require__(/*! ./modules/es6.reflect.get-own-property-descriptor */ \"./node_modules/core-js/modules/es6.reflect.get-own-property-descriptor.js\");\r\n__webpack_require__(/*! ./modules/es6.reflect.get-prototype-of */ \"./node_modules/core-js/modules/es6.reflect.get-prototype-of.js\");\r\n__webpack_require__(/*! ./modules/es6.reflect.has */ \"./node_modules/core-js/modules/es6.reflect.has.js\");\r\n__webpack_require__(/*! ./modules/es6.reflect.is-extensible */ \"./node_modules/core-js/modules/es6.reflect.is-extensible.js\");\r\n__webpack_require__(/*! ./modules/es6.reflect.own-keys */ \"./node_modules/core-js/modules/es6.reflect.own-keys.js\");\r\n__webpack_require__(/*! ./modules/es6.reflect.prevent-extensions */ \"./node_modules/core-js/modules/es6.reflect.prevent-extensions.js\");\r\n__webpack_require__(/*! ./modules/es6.reflect.set */ \"./node_modules/core-js/modules/es6.reflect.set.js\");\r\n__webpack_require__(/*! ./modules/es6.reflect.set-prototype-of */ \"./node_modules/core-js/modules/es6.reflect.set-prototype-of.js\");\r\n__webpack_require__(/*! ./modules/es7.array.includes */ \"./node_modules/core-js/modules/es7.array.includes.js\");\r\n__webpack_require__(/*! ./modules/es7.array.flat-map */ \"./node_modules/core-js/modules/es7.array.flat-map.js\");\r\n__webpack_require__(/*! ./modules/es7.array.flatten */ \"./node_modules/core-js/modules/es7.array.flatten.js\");\r\n__webpack_require__(/*! ./modules/es7.string.at */ \"./node_modules/core-js/modules/es7.string.at.js\");\r\n__webpack_require__(/*! ./modules/es7.string.pad-start */ \"./node_modules/core-js/modules/es7.string.pad-start.js\");\r\n__webpack_require__(/*! ./modules/es7.string.pad-end */ \"./node_modules/core-js/modules/es7.string.pad-end.js\");\r\n__webpack_require__(/*! ./modules/es7.string.trim-left */ \"./node_modules/core-js/modules/es7.string.trim-left.js\");\r\n__webpack_require__(/*! ./modules/es7.string.trim-right */ \"./node_modules/core-js/modules/es7.string.trim-right.js\");\r\n__webpack_require__(/*! ./modules/es7.string.match-all */ \"./node_modules/core-js/modules/es7.string.match-all.js\");\r\n__webpack_require__(/*! ./modules/es7.symbol.async-iterator */ \"./node_modules/core-js/modules/es7.symbol.async-iterator.js\");\r\n__webpack_require__(/*! ./modules/es7.symbol.observable */ \"./node_modules/core-js/modules/es7.symbol.observable.js\");\r\n__webpack_require__(/*! ./modules/es7.object.get-own-property-descriptors */ \"./node_modules/core-js/modules/es7.object.get-own-property-descriptors.js\");\r\n__webpack_require__(/*! ./modules/es7.object.values */ \"./node_modules/core-js/modules/es7.object.values.js\");\r\n__webpack_require__(/*! ./modules/es7.object.entries */ \"./node_modules/core-js/modules/es7.object.entries.js\");\r\n__webpack_require__(/*! ./modules/es7.object.define-getter */ \"./node_modules/core-js/modules/es7.object.define-getter.js\");\r\n__webpack_require__(/*! ./modules/es7.object.define-setter */ \"./node_modules/core-js/modules/es7.object.define-setter.js\");\r\n__webpack_require__(/*! ./modules/es7.object.lookup-getter */ \"./node_modules/core-js/modules/es7.object.lookup-getter.js\");\r\n__webpack_require__(/*! ./modules/es7.object.lookup-setter */ \"./node_modules/core-js/modules/es7.object.lookup-setter.js\");\r\n__webpack_require__(/*! ./modules/es7.map.to-json */ \"./node_modules/core-js/modules/es7.map.to-json.js\");\r\n__webpack_require__(/*! ./modules/es7.set.to-json */ \"./node_modules/core-js/modules/es7.set.to-json.js\");\r\n__webpack_require__(/*! ./modules/es7.map.of */ \"./node_modules/core-js/modules/es7.map.of.js\");\r\n__webpack_require__(/*! ./modules/es7.set.of */ \"./node_modules/core-js/modules/es7.set.of.js\");\r\n__webpack_require__(/*! ./modules/es7.weak-map.of */ \"./node_modules/core-js/modules/es7.weak-map.of.js\");\r\n__webpack_require__(/*! ./modules/es7.weak-set.of */ \"./node_modules/core-js/modules/es7.weak-set.of.js\");\r\n__webpack_require__(/*! ./modules/es7.map.from */ \"./node_modules/core-js/modules/es7.map.from.js\");\r\n__webpack_require__(/*! ./modules/es7.set.from */ \"./node_modules/core-js/modules/es7.set.from.js\");\r\n__webpack_require__(/*! ./modules/es7.weak-map.from */ \"./node_modules/core-js/modules/es7.weak-map.from.js\");\r\n__webpack_require__(/*! ./modules/es7.weak-set.from */ \"./node_modules/core-js/modules/es7.weak-set.from.js\");\r\n__webpack_require__(/*! ./modules/es7.global */ \"./node_modules/core-js/modules/es7.global.js\");\r\n__webpack_require__(/*! ./modules/es7.system.global */ \"./node_modules/core-js/modules/es7.system.global.js\");\r\n__webpack_require__(/*! ./modules/es7.error.is-error */ \"./node_modules/core-js/modules/es7.error.is-error.js\");\r\n__webpack_require__(/*! ./modules/es7.math.clamp */ \"./node_modules/core-js/modules/es7.math.clamp.js\");\r\n__webpack_require__(/*! ./modules/es7.math.deg-per-rad */ \"./node_modules/core-js/modules/es7.math.deg-per-rad.js\");\r\n__webpack_require__(/*! ./modules/es7.math.degrees */ \"./node_modules/core-js/modules/es7.math.degrees.js\");\r\n__webpack_require__(/*! ./modules/es7.math.fscale */ \"./node_modules/core-js/modules/es7.math.fscale.js\");\r\n__webpack_require__(/*! ./modules/es7.math.iaddh */ \"./node_modules/core-js/modules/es7.math.iaddh.js\");\r\n__webpack_require__(/*! ./modules/es7.math.isubh */ \"./node_modules/core-js/modules/es7.math.isubh.js\");\r\n__webpack_require__(/*! ./modules/es7.math.imulh */ \"./node_modules/core-js/modules/es7.math.imulh.js\");\r\n__webpack_require__(/*! ./modules/es7.math.rad-per-deg */ \"./node_modules/core-js/modules/es7.math.rad-per-deg.js\");\r\n__webpack_require__(/*! ./modules/es7.math.radians */ \"./node_modules/core-js/modules/es7.math.radians.js\");\r\n__webpack_require__(/*! ./modules/es7.math.scale */ \"./node_modules/core-js/modules/es7.math.scale.js\");\r\n__webpack_require__(/*! ./modules/es7.math.umulh */ \"./node_modules/core-js/modules/es7.math.umulh.js\");\r\n__webpack_require__(/*! ./modules/es7.math.signbit */ \"./node_modules/core-js/modules/es7.math.signbit.js\");\r\n__webpack_require__(/*! ./modules/es7.promise.finally */ \"./node_modules/core-js/modules/es7.promise.finally.js\");\r\n__webpack_require__(/*! ./modules/es7.promise.try */ \"./node_modules/core-js/modules/es7.promise.try.js\");\r\n__webpack_require__(/*! ./modules/es7.reflect.define-metadata */ \"./node_modules/core-js/modules/es7.reflect.define-metadata.js\");\r\n__webpack_require__(/*! ./modules/es7.reflect.delete-metadata */ \"./node_modules/core-js/modules/es7.reflect.delete-metadata.js\");\r\n__webpack_require__(/*! ./modules/es7.reflect.get-metadata */ \"./node_modules/core-js/modules/es7.reflect.get-metadata.js\");\r\n__webpack_require__(/*! ./modules/es7.reflect.get-metadata-keys */ \"./node_modules/core-js/modules/es7.reflect.get-metadata-keys.js\");\r\n__webpack_require__(/*! ./modules/es7.reflect.get-own-metadata */ \"./node_modules/core-js/modules/es7.reflect.get-own-metadata.js\");\r\n__webpack_require__(/*! ./modules/es7.reflect.get-own-metadata-keys */ \"./node_modules/core-js/modules/es7.reflect.get-own-metadata-keys.js\");\r\n__webpack_require__(/*! ./modules/es7.reflect.has-metadata */ \"./node_modules/core-js/modules/es7.reflect.has-metadata.js\");\r\n__webpack_require__(/*! ./modules/es7.reflect.has-own-metadata */ \"./node_modules/core-js/modules/es7.reflect.has-own-metadata.js\");\r\n__webpack_require__(/*! ./modules/es7.reflect.metadata */ \"./node_modules/core-js/modules/es7.reflect.metadata.js\");\r\n__webpack_require__(/*! ./modules/es7.asap */ \"./node_modules/core-js/modules/es7.asap.js\");\r\n__webpack_require__(/*! ./modules/es7.observable */ \"./node_modules/core-js/modules/es7.observable.js\");\r\n__webpack_require__(/*! ./modules/web.timers */ \"./node_modules/core-js/modules/web.timers.js\");\r\n__webpack_require__(/*! ./modules/web.immediate */ \"./node_modules/core-js/modules/web.immediate.js\");\r\n__webpack_require__(/*! ./modules/web.dom.iterable */ \"./node_modules/core-js/modules/web.dom.iterable.js\");\r\nmodule.exports = __webpack_require__(/*! ./modules/_core */ \"./node_modules/core-js/modules/_core.js\");\r\n\n\n//# sourceURL=webpack://state-update-dialog/./node_modules/core-js/shim.js?");

/***/ })

/******/ 	});
/************************************************************************/
/******/ 	// The module cache
/******/ 	var __webpack_module_cache__ = {};
/******/ 	
/******/ 	// The require function
/******/ 	function __webpack_require__(moduleId) {
/******/ 		// Check if module is in cache
/******/ 		var cachedModule = __webpack_module_cache__[moduleId];
/******/ 		if (cachedModule !== undefined) {
/******/ 			return cachedModule.exports;
/******/ 		}
/******/ 		// Create a new module (and put it into the cache)
/******/ 		var module = __webpack_module_cache__[moduleId] = {
/******/ 			// no module.id needed
/******/ 			// no module.loaded needed
/******/ 			exports: {}
/******/ 		};
/******/ 	
/******/ 		// Execute the module function
/******/ 		__webpack_modules__[moduleId].call(module.exports, module, module.exports, __webpack_require__);
/******/ 	
/******/ 		// Return the exports of the module
/******/ 		return module.exports;
/******/ 	}
/******/ 	
/************************************************************************/
/******/ 	/* webpack/runtime/global */
/******/ 	!function() {
/******/ 		__webpack_require__.g = (function() {
/******/ 			if (typeof globalThis === 'object') return globalThis;
/******/ 			try {
/******/ 				return this || new Function('return this')();
/******/ 			} catch (e) {
/******/ 				if (typeof window === 'object') return window;
/******/ 			}
/******/ 		})();
/******/ 	}();
/******/ 	
/************************************************************************/
/******/ 	
/******/ 	// startup
/******/ 	// Load entry module and return exports
/******/ 	// This entry module can't be inlined because the eval devtool is used.
/******/ 	__webpack_require__("./node_modules/babel-polyfill/lib/index.js");
/******/ 	var __webpack_exports__ = __webpack_require__("./main.js");
/******/ 	
/******/ })()
;