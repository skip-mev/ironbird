import {
  Any,
  Message,
  MethodIdempotency,
  MethodKind,
  createRegistry,
  proto3,
  protoBase64
} from "./chunk-TNC6V2Y3.js";

// node_modules/@bufbuild/connect/dist/esm/code.js
var Code;
(function(Code2) {
  Code2[Code2["Canceled"] = 1] = "Canceled";
  Code2[Code2["Unknown"] = 2] = "Unknown";
  Code2[Code2["InvalidArgument"] = 3] = "InvalidArgument";
  Code2[Code2["DeadlineExceeded"] = 4] = "DeadlineExceeded";
  Code2[Code2["NotFound"] = 5] = "NotFound";
  Code2[Code2["AlreadyExists"] = 6] = "AlreadyExists";
  Code2[Code2["PermissionDenied"] = 7] = "PermissionDenied";
  Code2[Code2["ResourceExhausted"] = 8] = "ResourceExhausted";
  Code2[Code2["FailedPrecondition"] = 9] = "FailedPrecondition";
  Code2[Code2["Aborted"] = 10] = "Aborted";
  Code2[Code2["OutOfRange"] = 11] = "OutOfRange";
  Code2[Code2["Unimplemented"] = 12] = "Unimplemented";
  Code2[Code2["Internal"] = 13] = "Internal";
  Code2[Code2["Unavailable"] = 14] = "Unavailable";
  Code2[Code2["DataLoss"] = 15] = "DataLoss";
  Code2[Code2["Unauthenticated"] = 16] = "Unauthenticated";
})(Code || (Code = {}));

// node_modules/@bufbuild/connect/dist/esm/protocol-connect/code-string.js
function codeToString(value) {
  const name = Code[value];
  if (typeof name != "string") {
    return value.toString();
  }
  return name[0].toLowerCase() + name.substring(1).replace(/[A-Z]/g, (c) => "_" + c.toLowerCase());
}
var stringToCode;
function codeFromString(value) {
  if (!stringToCode) {
    stringToCode = {};
    for (const value2 of Object.values(Code)) {
      if (typeof value2 == "string") {
        continue;
      }
      stringToCode[codeToString(value2)] = value2;
    }
  }
  return stringToCode[value];
}

// node_modules/@bufbuild/connect/dist/esm/connect-error.js
var ConnectError = class _ConnectError extends Error {
  /**
   * Create a new ConnectError.
   * If no code is provided, code "unknown" is used.
   * Outgoing details are only relevant for the server side - a service may
   * raise an error with details, and it is up to the protocol implementation
   * to encode and send the details along with error.
   */
  constructor(message, code = Code.Unknown, metadata, outgoingDetails, cause) {
    super(createMessage(message, code));
    this.name = "ConnectError";
    Object.setPrototypeOf(this, new.target.prototype);
    this.rawMessage = message;
    this.code = code;
    this.metadata = new Headers(metadata !== null && metadata !== void 0 ? metadata : {});
    this.details = outgoingDetails !== null && outgoingDetails !== void 0 ? outgoingDetails : [];
    this.cause = cause;
  }
  /**
   * Convert any value - typically a caught error into a ConnectError,
   * following these rules:
   * - If the value is already a ConnectError, return it as is.
   * - If the value is an AbortError from the fetch API, return the message
   *   of the AbortError with code Canceled.
   * - For other Errors, return the error message with code Unknown by default.
   * - For other values, return the values String representation as a message,
   *   with the code Unknown by default.
   * The original value will be used for the "cause" property for the new
   * ConnectError.
   */
  static from(reason, code = Code.Unknown) {
    if (reason instanceof _ConnectError) {
      return reason;
    }
    if (reason instanceof Error) {
      if (reason.name == "AbortError") {
        return new _ConnectError(reason.message, Code.Canceled);
      }
      return new _ConnectError(reason.message, code, void 0, void 0, reason);
    }
    return new _ConnectError(String(reason), code, void 0, void 0, reason);
  }
  findDetails(typeOrRegistry) {
    const registry = "typeName" in typeOrRegistry ? {
      findMessage: (typeName) => typeName === typeOrRegistry.typeName ? typeOrRegistry : void 0
    } : typeOrRegistry;
    const details = [];
    for (const data of this.details) {
      if (data instanceof Message) {
        if (registry.findMessage(data.getType().typeName)) {
          details.push(data);
        }
        continue;
      }
      const type = registry.findMessage(data.type);
      if (type) {
        try {
          details.push(type.fromBinary(data.value));
        } catch (_) {
        }
      }
    }
    return details;
  }
};
function connectErrorDetails(error, typeOrRegistry, ...moreTypes) {
  const types = "typeName" in typeOrRegistry ? [typeOrRegistry, ...moreTypes] : [];
  const registry = "typeName" in typeOrRegistry ? createRegistry(...types) : typeOrRegistry;
  const details = [];
  for (const data of error.details) {
    if (data instanceof Message) {
      if (registry.findMessage(data.getType().typeName)) {
        details.push(data);
      }
      continue;
    }
    const type = registry.findMessage(data.type);
    if (type) {
      try {
        details.push(type.fromBinary(data.value));
      } catch (_) {
      }
    }
  }
  return details;
}
function createMessage(message, code) {
  return message.length ? `[${codeToString(code)}] ${message}` : `[${codeToString(code)}]`;
}
function connectErrorFromReason(reason, code = Code.Unknown) {
  if (reason instanceof ConnectError) {
    return reason;
  }
  if (reason instanceof Error) {
    if (reason.name == "AbortError") {
      return new ConnectError(reason.message, Code.Canceled);
    }
    return new ConnectError(reason.message, code);
  }
  return new ConnectError(String(reason), code);
}

// node_modules/@bufbuild/connect/dist/esm/http-headers.js
function encodeBinaryHeader(value) {
  let bytes;
  if (value instanceof Message) {
    bytes = value.toBinary();
  } else if (typeof value == "string") {
    bytes = new TextEncoder().encode(value);
  } else {
    bytes = value instanceof Uint8Array ? value : new Uint8Array(value);
  }
  return protoBase64.enc(bytes).replace(/=+$/, "");
}
function decodeBinaryHeader(value, type, options) {
  try {
    const bytes = protoBase64.dec(value);
    if (type) {
      return type.fromBinary(bytes, options);
    }
    return bytes;
  } catch (e) {
    throw ConnectError.from(e, Code.DataLoss);
  }
}
function appendHeaders(...headers) {
  const h = new Headers();
  for (const e of headers) {
    e.forEach((value, key) => {
      h.append(key, value);
    });
  }
  return h;
}

// node_modules/@bufbuild/connect/dist/esm/any-client.js
function makeAnyClient(service, createMethod) {
  const client = {};
  for (const [localName, methodInfo] of Object.entries(service.methods)) {
    const method = createMethod(Object.assign(Object.assign({}, methodInfo), {
      localName,
      service
    }));
    if (method != null) {
      client[localName] = method;
    }
  }
  return client;
}

// node_modules/@bufbuild/connect/dist/esm/protocol/compression.js
var compressedFlag = 1;
function compressionNegotiate(available, requested, accepted, headerNameAcceptEncoding) {
  let request = null;
  let response = null;
  let error = void 0;
  if (requested !== null && requested !== "identity") {
    const found = available.find((c) => c.name === requested);
    if (found) {
      request = found;
    } else {
      const acceptable = available.map((c) => c.name).join(",");
      error = new ConnectError(`unknown compression "${requested}": supported encodings are ${acceptable}`, Code.Unimplemented, {
        [headerNameAcceptEncoding]: acceptable
      });
    }
  }
  if (accepted === null || accepted === "") {
    response = request;
  } else {
    const acceptNames = accepted.split(",").map((n) => n.trim());
    for (const name of acceptNames) {
      const found = available.find((c) => c.name === name);
      if (found) {
        response = found;
        break;
      }
    }
  }
  return { request, response, error };
}

// node_modules/@bufbuild/connect/dist/esm/protocol/envelope.js
function createEnvelopeReadableStream(stream) {
  let reader;
  let buffer = new Uint8Array(0);
  function append(chunk) {
    const n = new Uint8Array(buffer.length + chunk.length);
    n.set(buffer);
    n.set(chunk, buffer.length);
    buffer = n;
  }
  return new ReadableStream({
    start() {
      reader = stream.getReader();
    },
    async pull(controller) {
      let header = void 0;
      for (; ; ) {
        if (header === void 0 && buffer.byteLength >= 5) {
          let length = 0;
          for (let i = 1; i < 5; i++) {
            length = (length << 8) + buffer[i];
          }
          header = { flags: buffer[0], length };
        }
        if (header !== void 0 && buffer.byteLength >= header.length + 5) {
          break;
        }
        const result = await reader.read();
        if (result.done) {
          break;
        }
        append(result.value);
      }
      if (header === void 0) {
        if (buffer.byteLength == 0) {
          controller.close();
          return;
        }
        controller.error(new ConnectError("premature end of stream", Code.DataLoss));
        return;
      }
      const data = buffer.subarray(5, 5 + header.length);
      buffer = buffer.subarray(5 + header.length);
      controller.enqueue({
        flags: header.flags,
        data
      });
    }
  });
}
async function envelopeCompress(envelope, compression, compressMinBytes) {
  let { flags, data } = envelope;
  if ((flags & compressedFlag) === compressedFlag) {
    throw new ConnectError("invalid envelope, already compressed", Code.Internal);
  }
  if (compression && data.byteLength >= compressMinBytes) {
    data = await compression.compress(data);
    flags = flags | compressedFlag;
  }
  return { data, flags };
}
async function envelopeDecompress(envelope, compression, readMaxBytes) {
  let { flags, data } = envelope;
  if ((flags & compressedFlag) === compressedFlag) {
    if (!compression) {
      throw new ConnectError("received compressed envelope, but do not know how to decompress", Code.InvalidArgument);
    }
    data = await compression.decompress(data, readMaxBytes);
    flags = flags ^ compressedFlag;
  }
  return { data, flags };
}
function encodeEnvelope(flags, data) {
  const bytes = new Uint8Array(data.length + 5);
  bytes.set(data, 5);
  const v = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
  v.setUint8(0, flags);
  v.setUint32(1, data.length);
  return bytes;
}

// node_modules/@bufbuild/connect/dist/esm/protocol/limit-io.js
var maxReadMaxBytes = 4294967295;
var maxWriteMaxBytes = maxReadMaxBytes;
var defaultCompressMinBytes = 1024;
function validateReadWriteMaxBytes(readMaxBytes, writeMaxBytes, compressMinBytes) {
  writeMaxBytes !== null && writeMaxBytes !== void 0 ? writeMaxBytes : writeMaxBytes = maxWriteMaxBytes;
  readMaxBytes !== null && readMaxBytes !== void 0 ? readMaxBytes : readMaxBytes = maxReadMaxBytes;
  compressMinBytes !== null && compressMinBytes !== void 0 ? compressMinBytes : compressMinBytes = defaultCompressMinBytes;
  if (writeMaxBytes < 1 || writeMaxBytes > maxWriteMaxBytes) {
    throw new ConnectError(`writeMaxBytes ${writeMaxBytes} must be >= 1 and <= ${maxWriteMaxBytes}`, Code.Internal);
  }
  if (readMaxBytes < 1 || readMaxBytes > maxReadMaxBytes) {
    throw new ConnectError(`readMaxBytes ${readMaxBytes} must be >= 1 and <= ${maxReadMaxBytes}`, Code.Internal);
  }
  return {
    readMaxBytes,
    writeMaxBytes,
    compressMinBytes
  };
}
function assertWriteMaxBytes(writeMaxBytes, bytesWritten) {
  if (bytesWritten > writeMaxBytes) {
    throw new ConnectError(`message size ${bytesWritten} is larger than configured writeMaxBytes ${writeMaxBytes}`, Code.ResourceExhausted);
  }
}
function assertReadMaxBytes(readMaxBytes, bytesRead, totalSizeKnown = false) {
  if (bytesRead > readMaxBytes) {
    let message = `message size is larger than configured readMaxBytes ${readMaxBytes}`;
    if (totalSizeKnown) {
      message = `message size ${bytesRead} is larger than configured readMaxBytes ${readMaxBytes}`;
    }
    throw new ConnectError(message, Code.ResourceExhausted);
  }
}

// node_modules/@bufbuild/connect/dist/esm/protocol/async-iterable.js
var __asyncValues = function(o) {
  if (!Symbol.asyncIterator)
    throw new TypeError("Symbol.asyncIterator is not defined.");
  var m = o[Symbol.asyncIterator], i;
  return m ? m.call(o) : (o = typeof __values === "function" ? __values(o) : o[Symbol.iterator](), i = {}, verb("next"), verb("throw"), verb("return"), i[Symbol.asyncIterator] = function() {
    return this;
  }, i);
  function verb(n) {
    i[n] = o[n] && function(v) {
      return new Promise(function(resolve, reject) {
        v = o[n](v), settle(resolve, reject, v.done, v.value);
      });
    };
  }
  function settle(resolve, reject, d, v) {
    Promise.resolve(v).then(function(v2) {
      resolve({ value: v2, done: d });
    }, reject);
  }
};
var __await = function(v) {
  return this instanceof __await ? (this.v = v, this) : new __await(v);
};
var __asyncGenerator = function(thisArg, _arguments, generator) {
  if (!Symbol.asyncIterator)
    throw new TypeError("Symbol.asyncIterator is not defined.");
  var g = generator.apply(thisArg, _arguments || []), i, q = [];
  return i = {}, verb("next"), verb("throw"), verb("return"), i[Symbol.asyncIterator] = function() {
    return this;
  }, i;
  function verb(n) {
    if (g[n])
      i[n] = function(v) {
        return new Promise(function(a, b) {
          q.push([n, v, a, b]) > 1 || resume(n, v);
        });
      };
  }
  function resume(n, v) {
    try {
      step(g[n](v));
    } catch (e) {
      settle(q[0][3], e);
    }
  }
  function step(r) {
    r.value instanceof __await ? Promise.resolve(r.value.v).then(fulfill, reject) : settle(q[0][2], r);
  }
  function fulfill(value) {
    resume("next", value);
  }
  function reject(value) {
    resume("throw", value);
  }
  function settle(f, v) {
    if (f(v), q.shift(), q.length)
      resume(q[0][0], q[0][1]);
  }
};
var __asyncDelegator = function(o) {
  var i, p;
  return i = {}, verb("next"), verb("throw", function(e) {
    throw e;
  }), verb("return"), i[Symbol.iterator] = function() {
    return this;
  }, i;
  function verb(n, f) {
    i[n] = o[n] ? function(v) {
      return (p = !p) ? { value: __await(o[n](v)), done: false } : f ? f(v) : v;
    } : f;
  }
};
function pipeTo(source, ...rest) {
  const [transforms, sink, opt] = pickTransformsAndSink(rest);
  let iterable = source;
  let abortable;
  if ((opt === null || opt === void 0 ? void 0 : opt.propagateDownStreamError) === true) {
    iterable = abortable = makeIterableAbortable(iterable);
  }
  iterable = pipe(iterable, ...transforms, { propagateDownStreamError: false });
  return sink(iterable).catch((reason) => {
    if (abortable) {
      return abortable.abort(reason).then(() => Promise.reject(reason));
    }
    return Promise.reject(reason);
  });
}
function pickTransformsAndSink(rest) {
  let opt;
  if (typeof rest[rest.length - 1] != "function") {
    opt = rest.pop();
  }
  const sink = rest.pop();
  return [rest, sink, opt];
}
function sinkAllBytes(readMaxBytes, lengthHint) {
  return async function(iterable) {
    return await readAllBytes(iterable, readMaxBytes, lengthHint);
  };
}
function pipe(source, ...rest) {
  var _a, _b;
  return __asyncGenerator(this, arguments, function* pipe_1() {
    const [transforms, opt] = pickTransforms(rest);
    let abortable;
    let iterable = source;
    if ((opt === null || opt === void 0 ? void 0 : opt.propagateDownStreamError) === true) {
      iterable = abortable = makeIterableAbortable(iterable);
    }
    for (const t of transforms) {
      iterable = t(iterable);
    }
    const it = iterable[Symbol.asyncIterator]();
    try {
      for (; ; ) {
        const r = yield __await(it.next());
        if (r.done === true) {
          break;
        }
        if (!abortable) {
          yield yield __await(r.value);
          continue;
        }
        try {
          yield yield __await(r.value);
        } catch (e) {
          yield __await(abortable.abort(e));
          throw e;
        }
      }
    } finally {
      if ((opt === null || opt === void 0 ? void 0 : opt.propagateDownStreamError) === true) {
        (_b = (_a = source[Symbol.asyncIterator]()).return) === null || _b === void 0 ? void 0 : _b.call(_a).catch(() => {
        });
      }
    }
  });
}
function pickTransforms(rest) {
  let opt;
  if (typeof rest[rest.length - 1] != "function") {
    opt = rest.pop();
  }
  return [rest, opt];
}
function transformCatchFinally(catchFinally) {
  return function(iterable) {
    return __asyncGenerator(this, arguments, function* () {
      let err;
      const it = iterable[Symbol.asyncIterator]();
      for (; ; ) {
        let r;
        try {
          r = yield __await(it.next());
        } catch (e) {
          err = e;
          break;
        }
        if (r.done === true) {
          break;
        }
        yield yield __await(r.value);
      }
      const caught = yield __await(catchFinally(err));
      if (caught !== void 0) {
        yield yield __await(caught);
      }
    });
  };
}
function transformPrepend(provide) {
  return function(iterable) {
    return __asyncGenerator(this, arguments, function* () {
      var _a, e_3, _b, _c;
      const prepend = yield __await(provide());
      if (prepend !== void 0) {
        yield yield __await(prepend);
      }
      try {
        for (var _d = true, iterable_3 = __asyncValues(iterable), iterable_3_1; iterable_3_1 = yield __await(iterable_3.next()), _a = iterable_3_1.done, !_a; _d = true) {
          _c = iterable_3_1.value;
          _d = false;
          const chunk = _c;
          yield yield __await(chunk);
        }
      } catch (e_3_1) {
        e_3 = { error: e_3_1 };
      } finally {
        try {
          if (!_d && !_a && (_b = iterable_3.return))
            yield __await(_b.call(iterable_3));
        } finally {
          if (e_3)
            throw e_3.error;
        }
      }
    });
  };
}
function transformSerializeEnvelope(serialization, endStreamFlag2, endSerialization) {
  if (endStreamFlag2 === void 0 || endSerialization === void 0) {
    return function(iterable) {
      return __asyncGenerator(this, arguments, function* () {
        var _a, e_4, _b, _c;
        try {
          for (var _d = true, iterable_4 = __asyncValues(iterable), iterable_4_1; iterable_4_1 = yield __await(iterable_4.next()), _a = iterable_4_1.done, !_a; _d = true) {
            _c = iterable_4_1.value;
            _d = false;
            const chunk = _c;
            const data = serialization.serialize(chunk);
            yield yield __await({ flags: 0, data });
          }
        } catch (e_4_1) {
          e_4 = { error: e_4_1 };
        } finally {
          try {
            if (!_d && !_a && (_b = iterable_4.return))
              yield __await(_b.call(iterable_4));
          } finally {
            if (e_4)
              throw e_4.error;
          }
        }
      });
    };
  }
  return function(iterable) {
    return __asyncGenerator(this, arguments, function* () {
      var _a, e_5, _b, _c;
      try {
        for (var _d = true, iterable_5 = __asyncValues(iterable), iterable_5_1; iterable_5_1 = yield __await(iterable_5.next()), _a = iterable_5_1.done, !_a; _d = true) {
          _c = iterable_5_1.value;
          _d = false;
          const chunk = _c;
          let data;
          let flags = 0;
          if (chunk.end) {
            flags = flags | endStreamFlag2;
            data = endSerialization.serialize(chunk.value);
          } else {
            data = serialization.serialize(chunk.value);
          }
          yield yield __await({ flags, data });
        }
      } catch (e_5_1) {
        e_5 = { error: e_5_1 };
      } finally {
        try {
          if (!_d && !_a && (_b = iterable_5.return))
            yield __await(_b.call(iterable_5));
        } finally {
          if (e_5)
            throw e_5.error;
        }
      }
    });
  };
}
function transformParseEnvelope(serialization, endStreamFlag2, endSerialization) {
  if (endSerialization && endStreamFlag2 !== void 0) {
    return function(iterable) {
      return __asyncGenerator(this, arguments, function* () {
        var _a, e_6, _b, _c;
        try {
          for (var _d = true, iterable_6 = __asyncValues(iterable), iterable_6_1; iterable_6_1 = yield __await(iterable_6.next()), _a = iterable_6_1.done, !_a; _d = true) {
            _c = iterable_6_1.value;
            _d = false;
            const { flags, data } = _c;
            if ((flags & endStreamFlag2) === endStreamFlag2) {
              yield yield __await({ value: endSerialization.parse(data), end: true });
            } else {
              yield yield __await({ value: serialization.parse(data), end: false });
            }
          }
        } catch (e_6_1) {
          e_6 = { error: e_6_1 };
        } finally {
          try {
            if (!_d && !_a && (_b = iterable_6.return))
              yield __await(_b.call(iterable_6));
          } finally {
            if (e_6)
              throw e_6.error;
          }
        }
      });
    };
  }
  return function(iterable) {
    return __asyncGenerator(this, arguments, function* () {
      var _a, e_7, _b, _c;
      try {
        for (var _d = true, iterable_7 = __asyncValues(iterable), iterable_7_1; iterable_7_1 = yield __await(iterable_7.next()), _a = iterable_7_1.done, !_a; _d = true) {
          _c = iterable_7_1.value;
          _d = false;
          const { flags, data } = _c;
          if (endStreamFlag2 !== void 0 && (flags & endStreamFlag2) === endStreamFlag2) {
            if (endSerialization === null) {
              throw new ConnectError("unexpected end flag", Code.InvalidArgument);
            }
            continue;
          }
          yield yield __await(serialization.parse(data));
        }
      } catch (e_7_1) {
        e_7 = { error: e_7_1 };
      } finally {
        try {
          if (!_d && !_a && (_b = iterable_7.return))
            yield __await(_b.call(iterable_7));
        } finally {
          if (e_7)
            throw e_7.error;
        }
      }
    });
  };
}
function transformCompressEnvelope(compression, compressMinBytes) {
  return function(iterable) {
    return __asyncGenerator(this, arguments, function* () {
      var _a, e_8, _b, _c;
      try {
        for (var _d = true, iterable_8 = __asyncValues(iterable), iterable_8_1; iterable_8_1 = yield __await(iterable_8.next()), _a = iterable_8_1.done, !_a; _d = true) {
          _c = iterable_8_1.value;
          _d = false;
          const env = _c;
          yield yield __await(yield __await(envelopeCompress(env, compression, compressMinBytes)));
        }
      } catch (e_8_1) {
        e_8 = { error: e_8_1 };
      } finally {
        try {
          if (!_d && !_a && (_b = iterable_8.return))
            yield __await(_b.call(iterable_8));
        } finally {
          if (e_8)
            throw e_8.error;
        }
      }
    });
  };
}
function transformDecompressEnvelope(compression, readMaxBytes) {
  return function(iterable) {
    return __asyncGenerator(this, arguments, function* () {
      var _a, e_9, _b, _c;
      try {
        for (var _d = true, iterable_9 = __asyncValues(iterable), iterable_9_1; iterable_9_1 = yield __await(iterable_9.next()), _a = iterable_9_1.done, !_a; _d = true) {
          _c = iterable_9_1.value;
          _d = false;
          const env = _c;
          yield yield __await(yield __await(envelopeDecompress(env, compression, readMaxBytes)));
        }
      } catch (e_9_1) {
        e_9 = { error: e_9_1 };
      } finally {
        try {
          if (!_d && !_a && (_b = iterable_9.return))
            yield __await(_b.call(iterable_9));
        } finally {
          if (e_9)
            throw e_9.error;
        }
      }
    });
  };
}
function transformJoinEnvelopes() {
  return function(iterable) {
    return __asyncGenerator(this, arguments, function* () {
      var _a, e_10, _b, _c;
      try {
        for (var _d = true, iterable_10 = __asyncValues(iterable), iterable_10_1; iterable_10_1 = yield __await(iterable_10.next()), _a = iterable_10_1.done, !_a; _d = true) {
          _c = iterable_10_1.value;
          _d = false;
          const { flags, data } = _c;
          yield yield __await(encodeEnvelope(flags, data));
        }
      } catch (e_10_1) {
        e_10 = { error: e_10_1 };
      } finally {
        try {
          if (!_d && !_a && (_b = iterable_10.return))
            yield __await(_b.call(iterable_10));
        } finally {
          if (e_10)
            throw e_10.error;
        }
      }
    });
  };
}
function transformSplitEnvelope(readMaxBytes) {
  function append(buffer, chunk) {
    const n = new Uint8Array(buffer.byteLength + chunk.byteLength);
    n.set(buffer);
    n.set(chunk, buffer.length);
    return n;
  }
  function shiftEnvelope(buffer, header) {
    if (buffer.byteLength < 5 + header.length) {
      return [void 0, buffer];
    }
    return [
      { flags: header.flags, data: buffer.subarray(5, 5 + header.length) },
      buffer.subarray(5 + header.length)
    ];
  }
  function peekHeader(buffer) {
    if (buffer.byteLength < 5) {
      return void 0;
    }
    const view = new DataView(buffer.buffer, buffer.byteOffset, buffer.byteLength);
    const length = view.getUint32(1);
    const flags = view.getUint8(0);
    return { length, flags };
  }
  return function(iterable) {
    return __asyncGenerator(this, arguments, function* () {
      var _a, e_11, _b, _c;
      let buffer = new Uint8Array(0);
      try {
        for (var _d = true, iterable_11 = __asyncValues(iterable), iterable_11_1; iterable_11_1 = yield __await(iterable_11.next()), _a = iterable_11_1.done, !_a; _d = true) {
          _c = iterable_11_1.value;
          _d = false;
          const chunk = _c;
          buffer = append(buffer, chunk);
          for (; ; ) {
            const header = peekHeader(buffer);
            if (!header) {
              break;
            }
            assertReadMaxBytes(readMaxBytes, header.length, true);
            let env;
            [env, buffer] = shiftEnvelope(buffer, header);
            if (!env) {
              break;
            }
            yield yield __await(env);
          }
        }
      } catch (e_11_1) {
        e_11 = { error: e_11_1 };
      } finally {
        try {
          if (!_d && !_a && (_b = iterable_11.return))
            yield __await(_b.call(iterable_11));
        } finally {
          if (e_11)
            throw e_11.error;
        }
      }
      if (buffer.byteLength > 0) {
        const header = peekHeader(buffer);
        let message = "protocol error: incomplete envelope";
        if (header) {
          message = `protocol error: promised ${header.length} bytes in enveloped message, got ${buffer.byteLength - 5} bytes`;
        }
        throw new ConnectError(message, Code.InvalidArgument);
      }
    });
  };
}
async function readAllBytes(iterable, readMaxBytes, lengthHint) {
  var _a, e_12, _b, _c, _d, e_13, _e, _f;
  const [ok, hint] = parseLengthHint(lengthHint);
  if (ok) {
    if (hint > readMaxBytes) {
      assertReadMaxBytes(readMaxBytes, hint, true);
    }
    const buffer = new Uint8Array(hint);
    let offset2 = 0;
    try {
      for (var _g = true, iterable_12 = __asyncValues(iterable), iterable_12_1; iterable_12_1 = await iterable_12.next(), _a = iterable_12_1.done, !_a; _g = true) {
        _c = iterable_12_1.value;
        _g = false;
        const chunk = _c;
        if (offset2 + chunk.byteLength > hint) {
          throw new ConnectError(`protocol error: promised ${hint} bytes, received ${offset2 + chunk.byteLength}`, Code.InvalidArgument);
        }
        buffer.set(chunk, offset2);
        offset2 += chunk.byteLength;
      }
    } catch (e_12_1) {
      e_12 = { error: e_12_1 };
    } finally {
      try {
        if (!_g && !_a && (_b = iterable_12.return))
          await _b.call(iterable_12);
      } finally {
        if (e_12)
          throw e_12.error;
      }
    }
    if (offset2 < hint) {
      throw new ConnectError(`protocol error: promised ${hint} bytes, received ${offset2}`, Code.InvalidArgument);
    }
    return buffer;
  }
  const chunks = [];
  let count = 0;
  try {
    for (var _h = true, iterable_13 = __asyncValues(iterable), iterable_13_1; iterable_13_1 = await iterable_13.next(), _d = iterable_13_1.done, !_d; _h = true) {
      _f = iterable_13_1.value;
      _h = false;
      const chunk = _f;
      count += chunk.byteLength;
      assertReadMaxBytes(readMaxBytes, count);
      chunks.push(chunk);
    }
  } catch (e_13_1) {
    e_13 = { error: e_13_1 };
  } finally {
    try {
      if (!_h && !_d && (_e = iterable_13.return))
        await _e.call(iterable_13);
    } finally {
      if (e_13)
        throw e_13.error;
    }
  }
  const all = new Uint8Array(count);
  let offset = 0;
  for (let chunk = chunks.shift(); chunk; chunk = chunks.shift()) {
    all.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return all;
}
function parseLengthHint(lengthHint) {
  if (lengthHint === void 0 || lengthHint === null) {
    return [false, 0];
  }
  const n = typeof lengthHint == "string" ? parseInt(lengthHint, 10) : lengthHint;
  if (!Number.isSafeInteger(n) || n < 0) {
    return [false, n];
  }
  return [true, n];
}
async function untilFirst(iterable) {
  const it = iterable[Symbol.asyncIterator]();
  let first = await it.next();
  return {
    [Symbol.asyncIterator]() {
      const w = {
        async next() {
          if (first !== null) {
            const n = first;
            first = null;
            return n;
          }
          return await it.next();
        }
      };
      if (it.throw !== void 0) {
        w.throw = (e) => it.throw(e);
      }
      if (it.return !== void 0) {
        w.return = (value) => it.return(value);
      }
      return w;
    }
  };
}
function makeIterableAbortable(iterable) {
  const innerCandidate = iterable[Symbol.asyncIterator]();
  if (innerCandidate.throw === void 0) {
    throw new Error("AsyncIterable does not implement throw");
  }
  const inner = innerCandidate;
  let aborted;
  let resultPromise;
  let it = {
    next() {
      resultPromise = inner.next().finally(() => {
        resultPromise = void 0;
      });
      return resultPromise;
    },
    throw(e) {
      return inner.throw(e);
    }
  };
  if (innerCandidate.return !== void 0) {
    it = Object.assign(Object.assign({}, it), { return(value) {
      return inner.return(value);
    } });
  }
  let used = false;
  return {
    abort(reason) {
      if (aborted) {
        return aborted.state;
      }
      const f = () => {
        return inner.throw(reason).then((r) => r.done === true ? "completed" : "caught", () => "rethrown");
      };
      if (resultPromise) {
        aborted = { reason, state: resultPromise.then(f, f) };
        return aborted.state;
      }
      aborted = { reason, state: f() };
      return aborted.state;
    },
    [Symbol.asyncIterator]() {
      if (used) {
        throw new Error("AsyncIterable cannot be re-used");
      }
      used = true;
      return it;
    }
  };
}
function createAsyncIterable(items) {
  return __asyncGenerator(this, arguments, function* createAsyncIterable_1() {
    yield __await(yield* __asyncDelegator(__asyncValues(items)));
  });
}

// node_modules/@bufbuild/connect/dist/esm/callback-client.js
var __asyncValues2 = function(o) {
  if (!Symbol.asyncIterator)
    throw new TypeError("Symbol.asyncIterator is not defined.");
  var m = o[Symbol.asyncIterator], i;
  return m ? m.call(o) : (o = typeof __values === "function" ? __values(o) : o[Symbol.iterator](), i = {}, verb("next"), verb("throw"), verb("return"), i[Symbol.asyncIterator] = function() {
    return this;
  }, i);
  function verb(n) {
    i[n] = o[n] && function(v) {
      return new Promise(function(resolve, reject) {
        v = o[n](v), settle(resolve, reject, v.done, v.value);
      });
    };
  }
  function settle(resolve, reject, d, v) {
    Promise.resolve(v).then(function(v2) {
      resolve({ value: v2, done: d });
    }, reject);
  }
};
function createCallbackClient(service, transport) {
  return makeAnyClient(service, (method) => {
    switch (method.kind) {
      case MethodKind.Unary:
        return createUnaryFn(transport, service, method);
      case MethodKind.ServerStreaming:
        return createServerStreamingFn(transport, service, method);
      default:
        return null;
    }
  });
}
function createUnaryFn(transport, service, method) {
  return function(requestMessage, callback, options) {
    const abort = new AbortController();
    options = wrapSignal(abort, options);
    transport.unary(service, method, abort.signal, options.timeoutMs, options.headers, requestMessage).then((response) => {
      var _a, _b;
      (_a = options === null || options === void 0 ? void 0 : options.onHeader) === null || _a === void 0 ? void 0 : _a.call(options, response.header);
      (_b = options === null || options === void 0 ? void 0 : options.onTrailer) === null || _b === void 0 ? void 0 : _b.call(options, response.trailer);
      callback(void 0, response.message);
    }, (reason) => {
      const err = ConnectError.from(reason, Code.Internal);
      if (err.code === Code.Canceled && abort.signal.aborted) {
        return;
      }
      callback(err, new method.O());
    });
    return () => abort.abort();
  };
}
function createServerStreamingFn(transport, service, method) {
  return function(input, onResponse, onClose, options) {
    const abort = new AbortController();
    async function run() {
      var _a, e_1, _b, _c;
      var _d, _e;
      options = wrapSignal(abort, options);
      const response = await transport.stream(service, method, options.signal, options.timeoutMs, options.headers, createAsyncIterable([input]));
      (_d = options.onHeader) === null || _d === void 0 ? void 0 : _d.call(options, response.header);
      try {
        for (var _f = true, _g = __asyncValues2(response.message), _h; _h = await _g.next(), _a = _h.done, !_a; _f = true) {
          _c = _h.value;
          _f = false;
          const message = _c;
          onResponse(message);
        }
      } catch (e_1_1) {
        e_1 = { error: e_1_1 };
      } finally {
        try {
          if (!_f && !_a && (_b = _g.return))
            await _b.call(_g);
        } finally {
          if (e_1)
            throw e_1.error;
        }
      }
      (_e = options.onTrailer) === null || _e === void 0 ? void 0 : _e.call(options, response.trailer);
      onClose(void 0);
    }
    run().catch((reason) => {
      const err = ConnectError.from(reason, Code.Internal);
      if (err.code === Code.Canceled && abort.signal.aborted) {
        onClose(void 0);
      } else {
        onClose(err);
      }
    });
    return () => abort.abort();
  };
}
function wrapSignal(abort, options) {
  if (options === null || options === void 0 ? void 0 : options.signal) {
    options.signal.addEventListener("abort", () => abort.abort());
    if (options.signal.aborted) {
      abort.abort();
    }
  }
  return Object.assign(Object.assign({}, options), { signal: abort.signal });
}

// node_modules/@bufbuild/connect/dist/esm/promise-client.js
var __asyncValues3 = function(o) {
  if (!Symbol.asyncIterator)
    throw new TypeError("Symbol.asyncIterator is not defined.");
  var m = o[Symbol.asyncIterator], i;
  return m ? m.call(o) : (o = typeof __values === "function" ? __values(o) : o[Symbol.iterator](), i = {}, verb("next"), verb("throw"), verb("return"), i[Symbol.asyncIterator] = function() {
    return this;
  }, i);
  function verb(n) {
    i[n] = o[n] && function(v) {
      return new Promise(function(resolve, reject) {
        v = o[n](v), settle(resolve, reject, v.done, v.value);
      });
    };
  }
  function settle(resolve, reject, d, v) {
    Promise.resolve(v).then(function(v2) {
      resolve({ value: v2, done: d });
    }, reject);
  }
};
var __await2 = function(v) {
  return this instanceof __await2 ? (this.v = v, this) : new __await2(v);
};
var __asyncDelegator2 = function(o) {
  var i, p;
  return i = {}, verb("next"), verb("throw", function(e) {
    throw e;
  }), verb("return"), i[Symbol.iterator] = function() {
    return this;
  }, i;
  function verb(n, f) {
    i[n] = o[n] ? function(v) {
      return (p = !p) ? { value: __await2(o[n](v)), done: false } : f ? f(v) : v;
    } : f;
  }
};
var __asyncGenerator2 = function(thisArg, _arguments, generator) {
  if (!Symbol.asyncIterator)
    throw new TypeError("Symbol.asyncIterator is not defined.");
  var g = generator.apply(thisArg, _arguments || []), i, q = [];
  return i = {}, verb("next"), verb("throw"), verb("return"), i[Symbol.asyncIterator] = function() {
    return this;
  }, i;
  function verb(n) {
    if (g[n])
      i[n] = function(v) {
        return new Promise(function(a, b) {
          q.push([n, v, a, b]) > 1 || resume(n, v);
        });
      };
  }
  function resume(n, v) {
    try {
      step(g[n](v));
    } catch (e) {
      settle(q[0][3], e);
    }
  }
  function step(r) {
    r.value instanceof __await2 ? Promise.resolve(r.value.v).then(fulfill, reject) : settle(q[0][2], r);
  }
  function fulfill(value) {
    resume("next", value);
  }
  function reject(value) {
    resume("throw", value);
  }
  function settle(f, v) {
    if (f(v), q.shift(), q.length)
      resume(q[0][0], q[0][1]);
  }
};
function createPromiseClient(service, transport) {
  return makeAnyClient(service, (method) => {
    switch (method.kind) {
      case MethodKind.Unary:
        return createUnaryFn2(transport, service, method);
      case MethodKind.ServerStreaming:
        return createServerStreamingFn2(transport, service, method);
      case MethodKind.ClientStreaming:
        return createClientStreamingFn(transport, service, method);
      case MethodKind.BiDiStreaming:
        return createBiDiStreamingFn(transport, service, method);
      default:
        return null;
    }
  });
}
function createUnaryFn2(transport, service, method) {
  return async function(input, options) {
    var _a, _b;
    const response = await transport.unary(service, method, options === null || options === void 0 ? void 0 : options.signal, options === null || options === void 0 ? void 0 : options.timeoutMs, options === null || options === void 0 ? void 0 : options.headers, input);
    (_a = options === null || options === void 0 ? void 0 : options.onHeader) === null || _a === void 0 ? void 0 : _a.call(options, response.header);
    (_b = options === null || options === void 0 ? void 0 : options.onTrailer) === null || _b === void 0 ? void 0 : _b.call(options, response.trailer);
    return response.message;
  };
}
function createServerStreamingFn2(transport, service, method) {
  return function(input, options) {
    return handleStreamResponse(transport.stream(service, method, options === null || options === void 0 ? void 0 : options.signal, options === null || options === void 0 ? void 0 : options.timeoutMs, options === null || options === void 0 ? void 0 : options.headers, createAsyncIterable([input])), options);
  };
}
function createClientStreamingFn(transport, service, method) {
  return async function(request, options) {
    var _a, e_1, _b, _c;
    var _d, _e;
    const response = await transport.stream(service, method, options === null || options === void 0 ? void 0 : options.signal, options === null || options === void 0 ? void 0 : options.timeoutMs, options === null || options === void 0 ? void 0 : options.headers, request);
    (_d = options === null || options === void 0 ? void 0 : options.onHeader) === null || _d === void 0 ? void 0 : _d.call(options, response.header);
    let singleMessage;
    try {
      for (var _f = true, _g = __asyncValues3(response.message), _h; _h = await _g.next(), _a = _h.done, !_a; _f = true) {
        _c = _h.value;
        _f = false;
        const message = _c;
        singleMessage = message;
      }
    } catch (e_1_1) {
      e_1 = { error: e_1_1 };
    } finally {
      try {
        if (!_f && !_a && (_b = _g.return))
          await _b.call(_g);
      } finally {
        if (e_1)
          throw e_1.error;
      }
    }
    if (!singleMessage) {
      throw new ConnectError("protocol error: missing response message", Code.Internal);
    }
    (_e = options === null || options === void 0 ? void 0 : options.onTrailer) === null || _e === void 0 ? void 0 : _e.call(options, response.trailer);
    return singleMessage;
  };
}
function createBiDiStreamingFn(transport, service, method) {
  return function(request, options) {
    return handleStreamResponse(transport.stream(service, method, options === null || options === void 0 ? void 0 : options.signal, options === null || options === void 0 ? void 0 : options.timeoutMs, options === null || options === void 0 ? void 0 : options.headers, request), options);
  };
}
function handleStreamResponse(stream, options) {
  const it = function() {
    var _a, _b;
    return __asyncGenerator2(this, arguments, function* () {
      const response = yield __await2(stream);
      (_a = options === null || options === void 0 ? void 0 : options.onHeader) === null || _a === void 0 ? void 0 : _a.call(options, response.header);
      yield __await2(yield* __asyncDelegator2(__asyncValues3(response.message)));
      (_b = options === null || options === void 0 ? void 0 : options.onTrailer) === null || _b === void 0 ? void 0 : _b.call(options, response.trailer);
    });
  }()[Symbol.asyncIterator]();
  return {
    [Symbol.asyncIterator]: () => ({
      next: () => it.next()
    })
  };
}

// node_modules/@bufbuild/connect/dist/esm/protocol/signals.js
function createLinkedAbortController(...signals) {
  const controller = new AbortController();
  const sa = signals.filter((s) => s !== void 0).concat(controller.signal);
  for (const signal of sa) {
    if (signal.aborted) {
      onAbort.apply(signal);
      break;
    }
    signal.addEventListener("abort", onAbort);
  }
  function onAbort() {
    if (!controller.signal.aborted) {
      controller.abort(getAbortSignalReason(this));
    }
    for (const signal of sa) {
      signal.removeEventListener("abort", onAbort);
    }
  }
  return controller;
}
function createDeadlineSignal(timeoutMs) {
  const controller = new AbortController();
  const listener = () => {
    controller.abort(new ConnectError("the operation timed out", Code.DeadlineExceeded));
  };
  let timeoutId;
  if (timeoutMs !== void 0) {
    if (timeoutMs <= 0)
      listener();
    else
      timeoutId = setTimeout(listener, timeoutMs);
  }
  return {
    signal: controller.signal,
    cleanup: () => clearTimeout(timeoutId)
  };
}
function getAbortSignalReason(signal) {
  if (!signal.aborted) {
    return void 0;
  }
  if (signal.reason !== void 0) {
    return signal.reason;
  }
  const e = new Error("This operation was aborted");
  e.name = "AbortError";
  return e;
}

// node_modules/@bufbuild/connect/dist/esm/implementation.js
function createHandlerContext(init) {
  let timeoutMs;
  if (init.timeoutMs !== void 0) {
    const date = new Date(Date.now() + init.timeoutMs);
    timeoutMs = () => date.getTime() - Date.now();
  } else {
    timeoutMs = () => void 0;
  }
  const deadline = createDeadlineSignal(init.timeoutMs);
  const abortController = createLinkedAbortController(deadline.signal, init.requestSignal, init.shutdownSignal);
  return Object.assign(Object.assign({}, init), { signal: abortController.signal, timeoutMs, requestHeader: new Headers(init.requestHeader), responseHeader: new Headers(init.responseHeader), responseTrailer: new Headers(init.responseTrailer), abort(reason) {
    deadline.cleanup();
    abortController.abort(reason);
  } });
}
function createMethodImplSpec(service, method, impl) {
  return {
    kind: method.kind,
    service,
    method,
    impl
  };
}
function createServiceImplSpec(service, impl) {
  const s = { service, methods: {} };
  for (const [localName, methodInfo] of Object.entries(service.methods)) {
    let fn = impl[localName];
    if (typeof fn == "function") {
      fn = fn.bind(impl);
    } else {
      const message = `${service.typeName}.${methodInfo.name} is not implemented`;
      fn = function unimplemented() {
        throw new ConnectError(message, Code.Unimplemented);
      };
    }
    s.methods[localName] = createMethodImplSpec(service, methodInfo, fn);
  }
  return s;
}

// node_modules/@bufbuild/connect/dist/esm/protocol-grpc-web/trailer.js
var trailerFlag = 128;
function trailerParse(data) {
  const headers = new Headers();
  const lines = new TextDecoder().decode(data).split("\r\n");
  for (const line of lines) {
    if (line === "") {
      continue;
    }
    const i = line.indexOf(":");
    if (i > 0) {
      const name = line.substring(0, i).trim();
      const value = line.substring(i + 1).trim();
      headers.append(name, value);
    }
  }
  return headers;
}
function trailerSerialize(trailer) {
  const lines = [];
  trailer.forEach((value, key) => {
    lines.push(`${key}: ${value}\r
`);
  });
  return new TextEncoder().encode(lines.join(""));
}
function createTrailerSerialization() {
  return {
    serialize: trailerSerialize,
    parse: trailerParse
  };
}

// node_modules/@bufbuild/connect/dist/esm/protocol-grpc/headers.js
var headerContentType = "Content-Type";
var headerEncoding = "Grpc-Encoding";
var headerAcceptEncoding = "Grpc-Accept-Encoding";
var headerTimeout = "Grpc-Timeout";
var headerGrpcStatus = "Grpc-Status";
var headerGrpcMessage = "Grpc-Message";
var headerStatusDetailsBin = "Grpc-Status-Details-Bin";
var headerMessageType = "Grpc-Message-Type";

// node_modules/@bufbuild/connect/dist/esm/protocol-grpc-web/headers.js
var headerXUserAgent = "X-User-Agent";
var headerXGrpcWeb = "X-Grpc-Web";

// node_modules/@bufbuild/connect/dist/esm/protocol-grpc-web/content-type.js
var contentTypeRegExp = /^application\/grpc-web(-text)?(?:\+(?:(json)(?:; ?charset=utf-?8)?|proto))?$/i;
var contentTypeProto = "application/grpc-web+proto";
var contentTypeJson = "application/grpc-web+json";
function parseContentType(contentType) {
  const match = contentType === null || contentType === void 0 ? void 0 : contentType.match(contentTypeRegExp);
  if (!match) {
    return void 0;
  }
  const text = !!match[1];
  const binary = !match[2];
  return { text, binary };
}

// node_modules/@bufbuild/connect/dist/esm/protocol-grpc/parse-timeout.js
function parseTimeout(value, maxTimeoutMs) {
  if (value === null) {
    return {};
  }
  const results = /^(\d{1,8})([HMSmun])$/.exec(value);
  if (results === null) {
    return {
      error: new ConnectError(`protocol error: invalid grpc timeout value: ${value}`, Code.InvalidArgument)
    };
  }
  const unitToMultiplicand = {
    H: 60 * 60 * 1e3,
    M: 60 * 1e3,
    S: 1e3,
    m: 1,
    u: 1e-3,
    n: 1e-6
    // nanosecond
  };
  const timeoutMs = unitToMultiplicand[results[2]] * parseInt(results[1]);
  if (timeoutMs > maxTimeoutMs) {
    return {
      timeoutMs,
      error: new ConnectError(`timeout ${timeoutMs}ms must be <= ${maxTimeoutMs}`, Code.InvalidArgument)
    };
  }
  return {
    timeoutMs
  };
}

// node_modules/@bufbuild/connect/dist/esm/protocol-grpc/gen/status_pb.js
var Status = class _Status extends Message {
  constructor(data) {
    super();
    this.code = 0;
    this.message = "";
    this.details = [];
    proto3.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _Status().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _Status().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _Status().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_Status, a, b);
  }
};
Status.runtime = proto3;
Status.typeName = "google.rpc.Status";
Status.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "code",
    kind: "scalar",
    T: 5
    /* ScalarType.INT32 */
  },
  {
    no: 2,
    name: "message",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  },
  { no: 3, name: "details", kind: "message", T: Any, repeated: true }
]);

// node_modules/@bufbuild/connect/dist/esm/protocol-grpc/trailer-status.js
var grpcStatusOk = "0";
function setTrailerStatus(target, error) {
  if (error) {
    target.set(headerGrpcStatus, error.code.toString(10));
    target.set(headerGrpcMessage, encodeURIComponent(error.rawMessage));
    if (error.details.length > 0) {
      const status = new Status({
        code: error.code,
        message: error.rawMessage,
        details: error.details.map((value) => value instanceof Message ? Any.pack(value) : new Any({
          typeUrl: `type.googleapis.com/${value.type}`,
          value: value.value
        }))
      });
      target.set(headerStatusDetailsBin, encodeBinaryHeader(status));
    }
  } else {
    target.set(headerGrpcStatus, grpcStatusOk.toString());
  }
  return target;
}
function findTrailerError(headerOrTrailer) {
  var _a;
  const statusBytes = headerOrTrailer.get(headerStatusDetailsBin);
  if (statusBytes != null) {
    const status = decodeBinaryHeader(statusBytes, Status);
    if (status.code == 0) {
      return void 0;
    }
    const error = new ConnectError(status.message, status.code, headerOrTrailer);
    error.details = status.details.map((any) => ({
      type: any.typeUrl.substring(any.typeUrl.lastIndexOf("/") + 1),
      value: any.value
    }));
    return error;
  }
  const grpcStatus = headerOrTrailer.get(headerGrpcStatus);
  if (grpcStatus != null) {
    if (grpcStatus === grpcStatusOk) {
      return void 0;
    }
    const code = parseInt(grpcStatus, 10);
    if (code in Code) {
      return new ConnectError(decodeURIComponent((_a = headerOrTrailer.get(headerGrpcMessage)) !== null && _a !== void 0 ? _a : ""), code, headerOrTrailer);
    }
    return new ConnectError(`invalid grpc-status: ${grpcStatus}`, Code.Internal, headerOrTrailer);
  }
  return void 0;
}

// node_modules/@bufbuild/connect/dist/esm/protocol/content-type-matcher.js
var contentTypeMatcherCacheSize = 1024;
function contentTypeMatcher(...supported) {
  const cache = /* @__PURE__ */ new Map();
  const source = supported.reduce((previousValue, currentValue) => previousValue.concat("supported" in currentValue ? currentValue.supported : currentValue), []);
  function match(contentType) {
    if (contentType === null || contentType.length == 0) {
      return false;
    }
    const cached = cache.get(contentType);
    if (cached !== void 0) {
      return cached;
    }
    const ok = source.some((re) => re.test(contentType));
    if (cache.size < contentTypeMatcherCacheSize) {
      cache.set(contentType, ok);
    }
    return ok;
  }
  match.supported = source;
  return match;
}

// node_modules/@bufbuild/connect/dist/esm/protocol/create-method-url.js
function createMethodUrl(baseUrl, service, method) {
  const s = typeof service == "string" ? service : service.typeName;
  const m = typeof method == "string" ? method : method.name;
  return baseUrl.toString().replace(/\/?$/, `/${s}/${m}`);
}

// node_modules/@bufbuild/connect/dist/esm/protocol/invoke-implementation.js
var __await3 = function(v) {
  return this instanceof __await3 ? (this.v = v, this) : new __await3(v);
};
var __asyncGenerator3 = function(thisArg, _arguments, generator) {
  if (!Symbol.asyncIterator)
    throw new TypeError("Symbol.asyncIterator is not defined.");
  var g = generator.apply(thisArg, _arguments || []), i, q = [];
  return i = {}, verb("next"), verb("throw"), verb("return"), i[Symbol.asyncIterator] = function() {
    return this;
  }, i;
  function verb(n) {
    if (g[n])
      i[n] = function(v) {
        return new Promise(function(a, b) {
          q.push([n, v, a, b]) > 1 || resume(n, v);
        });
      };
  }
  function resume(n, v) {
    try {
      step(g[n](v));
    } catch (e) {
      settle(q[0][3], e);
    }
  }
  function step(r) {
    r.value instanceof __await3 ? Promise.resolve(r.value.v).then(fulfill, reject) : settle(q[0][2], r);
  }
  function fulfill(value) {
    resume("next", value);
  }
  function reject(value) {
    resume("throw", value);
  }
  function settle(f, v) {
    if (f(v), q.shift(), q.length)
      resume(q[0][0], q[0][1]);
  }
};
var __asyncValues4 = function(o) {
  if (!Symbol.asyncIterator)
    throw new TypeError("Symbol.asyncIterator is not defined.");
  var m = o[Symbol.asyncIterator], i;
  return m ? m.call(o) : (o = typeof __values === "function" ? __values(o) : o[Symbol.iterator](), i = {}, verb("next"), verb("throw"), verb("return"), i[Symbol.asyncIterator] = function() {
    return this;
  }, i);
  function verb(n) {
    i[n] = o[n] && function(v) {
      return new Promise(function(resolve, reject) {
        v = o[n](v), settle(resolve, reject, v.done, v.value);
      });
    };
  }
  function settle(resolve, reject, d, v) {
    Promise.resolve(v).then(function(v2) {
      resolve({ value: v2, done: d });
    }, reject);
  }
};
async function invokeUnaryImplementation(spec, context, input) {
  const output = await spec.impl(input, context);
  return normalizeOutput(spec, output);
}
function transformInvokeImplementation(spec, context) {
  switch (spec.kind) {
    case MethodKind.Unary:
      return function unary(input) {
        return __asyncGenerator3(this, arguments, function* unary_1() {
          const inputIt = input[Symbol.asyncIterator]();
          const input1 = yield __await3(inputIt.next());
          if (input1.done === true) {
            throw new ConnectError("protocol error: missing input message for unary method", Code.InvalidArgument);
          }
          yield yield __await3(normalizeOutput(spec, yield __await3(spec.impl(input1.value, context))));
          const input2 = yield __await3(inputIt.next());
          if (input2.done !== true) {
            throw new ConnectError("protocol error: received extra input message for unary method", Code.InvalidArgument);
          }
        });
      };
    case MethodKind.ServerStreaming: {
      return function serverStreaming(input) {
        return __asyncGenerator3(this, arguments, function* serverStreaming_1() {
          var _a, e_1, _b, _c;
          const inputIt = input[Symbol.asyncIterator]();
          const input1 = yield __await3(inputIt.next());
          if (input1.done === true) {
            throw new ConnectError("protocol error: missing input message for server-streaming method", Code.InvalidArgument);
          }
          try {
            for (var _d = true, _e = __asyncValues4(spec.impl(input1.value, context)), _f; _f = yield __await3(_e.next()), _a = _f.done, !_a; _d = true) {
              _c = _f.value;
              _d = false;
              const o = _c;
              yield yield __await3(normalizeOutput(spec, o));
            }
          } catch (e_1_1) {
            e_1 = { error: e_1_1 };
          } finally {
            try {
              if (!_d && !_a && (_b = _e.return))
                yield __await3(_b.call(_e));
            } finally {
              if (e_1)
                throw e_1.error;
            }
          }
          const input2 = yield __await3(inputIt.next());
          if (input2.done !== true) {
            throw new ConnectError("protocol error: received extra input message for server-streaming method", Code.InvalidArgument);
          }
        });
      };
    }
    case MethodKind.ClientStreaming: {
      return function clientStreaming(input) {
        return __asyncGenerator3(this, arguments, function* clientStreaming_1() {
          yield yield __await3(normalizeOutput(spec, yield __await3(spec.impl(input, context))));
        });
      };
    }
    case MethodKind.BiDiStreaming:
      return function biDiStreaming(input) {
        return __asyncGenerator3(this, arguments, function* biDiStreaming_1() {
          var _a, e_2, _b, _c;
          try {
            for (var _d = true, _e = __asyncValues4(spec.impl(input, context)), _f; _f = yield __await3(_e.next()), _a = _f.done, !_a; _d = true) {
              _c = _f.value;
              _d = false;
              const o = _c;
              yield yield __await3(normalizeOutput(spec, o));
            }
          } catch (e_2_1) {
            e_2 = { error: e_2_1 };
          } finally {
            try {
              if (!_d && !_a && (_b = _e.return))
                yield __await3(_b.call(_e));
            } finally {
              if (e_2)
                throw e_2.error;
            }
          }
        });
      };
  }
}
function normalizeOutput(spec, message) {
  if (message instanceof Message) {
    return message;
  }
  try {
    return new spec.method.O(message);
  } catch (e) {
    throw new ConnectError(`failed to normalize message ${spec.method.O.typeName}`, Code.Internal, void 0, void 0, e);
  }
}

// node_modules/@bufbuild/connect/dist/esm/protocol/serialization.js
function getJsonOptions(options) {
  var _a;
  const o = Object.assign({}, options);
  (_a = o.ignoreUnknownFields) !== null && _a !== void 0 ? _a : o.ignoreUnknownFields = true;
  return o;
}
function createMethodSerializationLookup(method, binaryOptions, jsonOptions, limitOptions) {
  const inputBinary = limitSerialization(createBinarySerialization(method.I, binaryOptions), limitOptions);
  const inputJson = limitSerialization(createJsonSerialization(method.I, jsonOptions), limitOptions);
  const outputBinary = limitSerialization(createBinarySerialization(method.O, binaryOptions), limitOptions);
  const outputJson = limitSerialization(createJsonSerialization(method.O, jsonOptions), limitOptions);
  return {
    getI(useBinaryFormat) {
      return useBinaryFormat ? inputBinary : inputJson;
    },
    getO(useBinaryFormat) {
      return useBinaryFormat ? outputBinary : outputJson;
    }
  };
}
function createClientMethodSerializers(method, useBinaryFormat, jsonOptions, binaryOptions) {
  const input = useBinaryFormat ? createBinarySerialization(method.I, binaryOptions) : createJsonSerialization(method.I, jsonOptions);
  const output = useBinaryFormat ? createBinarySerialization(method.O, binaryOptions) : createJsonSerialization(method.O, jsonOptions);
  return { parse: output.parse, serialize: input.serialize };
}
function limitSerialization(serialization, limitOptions) {
  return {
    serialize(data) {
      const bytes = serialization.serialize(data);
      assertWriteMaxBytes(limitOptions.writeMaxBytes, bytes.byteLength);
      return bytes;
    },
    parse(data) {
      assertReadMaxBytes(limitOptions.readMaxBytes, data.byteLength, true);
      return serialization.parse(data);
    }
  };
}
function createBinarySerialization(messageType, options) {
  return {
    parse(data) {
      try {
        return messageType.fromBinary(data, options);
      } catch (e) {
        const m = e instanceof Error ? e.message : String(e);
        throw new ConnectError(`parse binary: ${m}`, Code.InvalidArgument);
      }
    },
    serialize(data) {
      try {
        return data.toBinary(options);
      } catch (e) {
        const m = e instanceof Error ? e.message : String(e);
        throw new ConnectError(`serialize binary: ${m}`, Code.Internal);
      }
    }
  };
}
function createJsonSerialization(messageType, options) {
  var _a, _b;
  const textEncoder = (_a = options === null || options === void 0 ? void 0 : options.textEncoder) !== null && _a !== void 0 ? _a : new TextEncoder();
  const textDecoder = (_b = options === null || options === void 0 ? void 0 : options.textDecoder) !== null && _b !== void 0 ? _b : new TextDecoder();
  const o = getJsonOptions(options);
  return {
    parse(data) {
      try {
        const json = textDecoder.decode(data);
        return messageType.fromJsonString(json, o);
      } catch (e) {
        throw ConnectError.from(e, Code.InvalidArgument);
      }
    },
    serialize(data) {
      try {
        const json = data.toJsonString(o);
        return textEncoder.encode(json);
      } catch (e) {
        throw ConnectError.from(e, Code.Internal);
      }
    }
  };
}

// node_modules/@bufbuild/connect/dist/esm/protocol/universal.js
function assertByteStreamRequest(req) {
  if (typeof req.body == "object" && req.body !== null && Symbol.asyncIterator in req.body) {
    return;
  }
  throw new Error("byte stream required, but received JSON");
}
var uResponseOk = {
  status: 200
};
var uResponseUnsupportedMediaType = {
  status: 415
};
var uResponseMethodNotAllowed = {
  status: 405
};
var uResponseVersionNotSupported = {
  status: 505
};

// node_modules/@bufbuild/connect/dist/esm/protocol/universal-handler.js
function validateUniversalHandlerOptions(opt) {
  var _a, _b;
  opt !== null && opt !== void 0 ? opt : opt = {};
  const acceptCompression = opt.acceptCompression ? [...opt.acceptCompression] : [];
  const requireConnectProtocolHeader = (_a = opt.requireConnectProtocolHeader) !== null && _a !== void 0 ? _a : false;
  const maxTimeoutMs = (_b = opt.maxTimeoutMs) !== null && _b !== void 0 ? _b : Number.MAX_SAFE_INTEGER;
  return Object.assign(Object.assign({ acceptCompression }, validateReadWriteMaxBytes(opt.readMaxBytes, opt.writeMaxBytes, opt.compressMinBytes)), { jsonOptions: opt.jsonOptions, binaryOptions: opt.binaryOptions, maxTimeoutMs, shutdownSignal: opt.shutdownSignal, requireConnectProtocolHeader });
}
function createUniversalServiceHandlers(spec, protocols) {
  return Object.entries(spec.methods).map(([, implSpec]) => createUniversalMethodHandler(implSpec, protocols));
}
function createUniversalMethodHandler(spec, protocols) {
  return negotiateProtocol(protocols.map((f) => f(spec)));
}
function negotiateProtocol(protocolHandlers) {
  if (protocolHandlers.length == 0) {
    throw new ConnectError("at least one protocol is required", Code.Internal);
  }
  const service = protocolHandlers[0].service;
  const method = protocolHandlers[0].method;
  const requestPath = protocolHandlers[0].requestPath;
  if (protocolHandlers.some((h) => h.service !== service || h.method !== method)) {
    throw new ConnectError("cannot negotiate protocol for different RPCs", Code.Internal);
  }
  if (protocolHandlers.some((h) => h.requestPath !== requestPath)) {
    throw new ConnectError("cannot negotiate protocol for different requestPaths", Code.Internal);
  }
  async function protocolNegotiatingHandler(request) {
    var _a;
    if (method.kind == MethodKind.BiDiStreaming && request.httpVersion.startsWith("1.")) {
      return Object.assign(Object.assign({}, uResponseVersionNotSupported), {
        // Clients coded to expect full-duplex connections may hang if they've
        // mistakenly negotiated HTTP/1.1. To unblock them, we must close the
        // underlying TCP connection.
        header: new Headers({ Connection: "close" })
      });
    }
    const contentType = (_a = request.header.get("Content-Type")) !== null && _a !== void 0 ? _a : "";
    const matchingMethod = protocolHandlers.filter((h) => h.allowedMethods.includes(request.method));
    if (matchingMethod.length == 0) {
      return uResponseMethodNotAllowed;
    }
    if (matchingMethod.length == 1 && contentType === "") {
      const onlyMatch = matchingMethod[0];
      return onlyMatch(request);
    }
    const matchingContentTypes = matchingMethod.filter((h) => h.supportedContentType(contentType));
    if (matchingContentTypes.length == 0) {
      return uResponseUnsupportedMediaType;
    }
    const firstMatch = matchingContentTypes[0];
    return firstMatch(request);
  }
  return Object.assign(protocolNegotiatingHandler, {
    service,
    method,
    requestPath,
    supportedContentType: contentTypeMatcher(...protocolHandlers.map((h) => h.supportedContentType)),
    protocolNames: protocolHandlers.flatMap((h) => h.protocolNames).filter((value, index, array) => array.indexOf(value) === index),
    allowedMethods: protocolHandlers.flatMap((h) => h.allowedMethods).filter((value, index, array) => array.indexOf(value) === index)
  });
}

// node_modules/@bufbuild/connect/dist/esm/protocol-grpc-web/handler-factory.js
var protocolName = "grpc-web";
var methodPost = "POST";
function createHandlerFactory(options) {
  const opt = validateUniversalHandlerOptions(options);
  const trailerSerialization = createTrailerSerialization();
  function fact(spec) {
    const h = createHandler(opt, trailerSerialization, spec);
    return Object.assign(h, {
      protocolNames: [protocolName],
      allowedMethods: [methodPost],
      supportedContentType: contentTypeMatcher(contentTypeRegExp),
      requestPath: createMethodUrl("/", spec.service, spec.method),
      service: spec.service,
      method: spec.method
    });
  }
  fact.protocolName = protocolName;
  return fact;
}
function createHandler(opt, trailerSerialization, spec) {
  const serialization = createMethodSerializationLookup(spec.method, opt.binaryOptions, opt.jsonOptions, opt);
  return async function handle(req) {
    assertByteStreamRequest(req);
    const type = parseContentType(req.header.get(headerContentType));
    if (type == void 0 || type.text) {
      return uResponseUnsupportedMediaType;
    }
    if (req.method !== methodPost) {
      return uResponseMethodNotAllowed;
    }
    const timeout = parseTimeout(req.header.get(headerTimeout), opt.maxTimeoutMs);
    const context = createHandlerContext(Object.assign(Object.assign({}, spec), { requestMethod: req.method, protocolName, timeoutMs: timeout.timeoutMs, shutdownSignal: opt.shutdownSignal, requestSignal: req.signal, requestHeader: req.header, responseHeader: {
      [headerContentType]: type.binary ? contentTypeProto : contentTypeJson
    }, responseTrailer: {
      [headerGrpcStatus]: grpcStatusOk
    } }));
    const compression = compressionNegotiate(opt.acceptCompression, req.header.get(headerEncoding), req.header.get(headerAcceptEncoding), headerAcceptEncoding);
    if (compression.response) {
      context.responseHeader.set(headerEncoding, compression.response.name);
    }
    const outputIt = pipe(req.body, transformPrepend(() => {
      if (compression.error)
        throw compression.error;
      if (timeout.error)
        throw timeout.error;
      return void 0;
    }), transformSplitEnvelope(opt.readMaxBytes), transformDecompressEnvelope(compression.request, opt.readMaxBytes), transformParseEnvelope(serialization.getI(type.binary), trailerFlag), transformInvokeImplementation(spec, context), transformSerializeEnvelope(serialization.getO(type.binary)), transformCatchFinally((e) => {
      context.abort();
      if (e instanceof ConnectError) {
        setTrailerStatus(context.responseTrailer, e);
      } else if (e !== void 0) {
        setTrailerStatus(context.responseTrailer, new ConnectError("internal error", Code.Internal, void 0, void 0, e));
      }
      return {
        flags: trailerFlag,
        data: trailerSerialization.serialize(context.responseTrailer)
      };
    }), transformCompressEnvelope(compression.response, opt.compressMinBytes), transformJoinEnvelopes());
    return Object.assign(Object.assign({}, uResponseOk), {
      // We wait for the first response body bytes before resolving, so that
      // implementations have a chance to add headers before an adapter commits
      // them to the wire.
      body: await untilFirst(outputIt),
      header: context.responseHeader
    });
  };
}

// node_modules/@bufbuild/connect/dist/esm/protocol-grpc/content-type.js
var contentTypeRegExp2 = /^application\/grpc(?:\+(?:(json)(?:; ?charset=utf-?8)?|proto))?$/i;
var contentTypeProto2 = "application/grpc+proto";
var contentTypeJson2 = "application/grpc+json";
function parseContentType2(contentType) {
  const match = contentType === null || contentType === void 0 ? void 0 : contentType.match(contentTypeRegExp2);
  if (!match) {
    return void 0;
  }
  const binary = !match[1];
  return { binary };
}

// node_modules/@bufbuild/connect/dist/esm/protocol-grpc/handler-factory.js
var protocolName2 = "grpc";
var methodPost2 = "POST";
function createHandlerFactory2(options) {
  const opt = validateUniversalHandlerOptions(options);
  function fact(spec) {
    const h = createHandler2(opt, spec);
    return Object.assign(h, {
      protocolNames: [protocolName2],
      allowedMethods: [methodPost2],
      supportedContentType: contentTypeMatcher(contentTypeRegExp2),
      requestPath: createMethodUrl("/", spec.service, spec.method),
      service: spec.service,
      method: spec.method
    });
  }
  fact.protocolName = protocolName2;
  return fact;
}
function createHandler2(opt, spec) {
  const serialization = createMethodSerializationLookup(spec.method, opt.binaryOptions, opt.jsonOptions, opt);
  return async function handle(req) {
    assertByteStreamRequest(req);
    const type = parseContentType2(req.header.get(headerContentType));
    if (type == void 0) {
      return uResponseUnsupportedMediaType;
    }
    if (req.method !== methodPost2) {
      return uResponseMethodNotAllowed;
    }
    const timeout = parseTimeout(req.header.get(headerTimeout), opt.maxTimeoutMs);
    const context = createHandlerContext(Object.assign(Object.assign({}, spec), { requestMethod: req.method, protocolName: protocolName2, timeoutMs: timeout.timeoutMs, shutdownSignal: opt.shutdownSignal, requestSignal: req.signal, requestHeader: req.header, responseHeader: {
      [headerContentType]: type.binary ? contentTypeProto2 : contentTypeJson2
    }, responseTrailer: {
      [headerGrpcStatus]: grpcStatusOk
    } }));
    const compression = compressionNegotiate(opt.acceptCompression, req.header.get(headerEncoding), req.header.get(headerAcceptEncoding), headerAcceptEncoding);
    if (compression.response) {
      context.responseHeader.set(headerEncoding, compression.response.name);
    }
    const outputIt = pipe(req.body, transformPrepend(() => {
      if (compression.error)
        throw compression.error;
      if (timeout.error)
        throw timeout.error;
      return void 0;
    }), transformSplitEnvelope(opt.readMaxBytes), transformDecompressEnvelope(compression.request, opt.readMaxBytes), transformParseEnvelope(serialization.getI(type.binary)), transformInvokeImplementation(spec, context), transformSerializeEnvelope(serialization.getO(type.binary)), transformCompressEnvelope(compression.response, opt.compressMinBytes), transformJoinEnvelopes(), transformCatchFinally((e) => {
      context.abort();
      if (e instanceof ConnectError) {
        setTrailerStatus(context.responseTrailer, e);
      } else if (e !== void 0) {
        setTrailerStatus(context.responseTrailer, new ConnectError("internal error", Code.Internal, void 0, void 0, e));
      }
    }));
    return Object.assign(Object.assign({}, uResponseOk), {
      // We wait for the first response body bytes before resolving, so that
      // implementations have a chance to add headers before an adapter commits
      // them to the wire.
      body: await untilFirst(outputIt),
      header: context.responseHeader,
      trailer: context.responseTrailer
    });
  };
}

// node_modules/@bufbuild/connect/dist/esm/protocol-connect/content-type.js
var contentTypeRegExp3 = /^application\/(connect\+)?(?:(json)(?:; ?charset=utf-?8)?|(proto))$/i;
var contentTypeUnaryRegExp = /^application\/(?:json(?:; ?charset=utf-?8)?|proto)$/i;
var contentTypeStreamRegExp = /^application\/connect\+?(?:json(?:; ?charset=utf-?8)?|proto)$/i;
var contentTypeUnaryProto = "application/proto";
var contentTypeUnaryJson = "application/json";
var contentTypeStreamProto = "application/connect+proto";
var contentTypeStreamJson = "application/connect+json";
var encodingProto = "proto";
var encodingJson = "json";
function parseContentType3(contentType) {
  const match = contentType === null || contentType === void 0 ? void 0 : contentType.match(contentTypeRegExp3);
  if (!match) {
    return void 0;
  }
  const stream = !!match[1];
  const binary = !!match[3];
  return { stream, binary };
}
function parseEncodingQuery(encoding) {
  switch (encoding) {
    case encodingProto:
      return { stream: false, binary: true };
    case encodingJson:
      return { stream: false, binary: false };
    default:
      return void 0;
  }
}

// node_modules/@bufbuild/connect/dist/esm/protocol-connect/error-json.js
var __rest = function(s, e) {
  var t = {};
  for (var p in s)
    if (Object.prototype.hasOwnProperty.call(s, p) && e.indexOf(p) < 0)
      t[p] = s[p];
  if (s != null && typeof Object.getOwnPropertySymbols === "function")
    for (var i = 0, p = Object.getOwnPropertySymbols(s); i < p.length; i++) {
      if (e.indexOf(p[i]) < 0 && Object.prototype.propertyIsEnumerable.call(s, p[i]))
        t[p[i]] = s[p[i]];
    }
  return t;
};
function errorFromJson(jsonValue, metadata, fallback) {
  if (metadata) {
    new Headers(metadata).forEach((value, key) => fallback.metadata.append(key, value));
  }
  if (typeof jsonValue !== "object" || jsonValue == null || Array.isArray(jsonValue) || !("code" in jsonValue) || typeof jsonValue.code !== "string") {
    throw fallback;
  }
  const code = codeFromString(jsonValue.code);
  if (code === void 0) {
    throw fallback;
  }
  const message = jsonValue.message;
  if (message != null && typeof message !== "string") {
    throw fallback;
  }
  const error = new ConnectError(message !== null && message !== void 0 ? message : "", code, metadata);
  if ("details" in jsonValue && Array.isArray(jsonValue.details)) {
    for (const detail of jsonValue.details) {
      if (detail === null || typeof detail != "object" || Array.isArray(detail) || typeof detail.type != "string" || typeof detail.value != "string" || "debug" in detail && typeof detail.debug != "object") {
        throw fallback;
      }
      try {
        error.details.push({
          type: detail.type,
          value: protoBase64.dec(detail.value),
          debug: detail.debug
        });
      } catch (e) {
        throw fallback;
      }
    }
  }
  return error;
}
function errorFromJsonBytes(bytes, metadata, fallback) {
  let jsonValue;
  try {
    jsonValue = JSON.parse(new TextDecoder().decode(bytes));
  } catch (e) {
    throw fallback;
  }
  return errorFromJson(jsonValue, metadata, fallback);
}
function errorToJson(error, jsonWriteOptions) {
  const o = {
    code: codeToString(error.code)
  };
  if (error.rawMessage.length > 0) {
    o.message = error.rawMessage;
  }
  if (error.details.length > 0) {
    o.details = error.details.map((value) => {
      if (value instanceof Message) {
        const i = {
          type: value.getType().typeName,
          value: value.toBinary()
        };
        try {
          i.debug = value.toJson(jsonWriteOptions);
        } catch (e) {
        }
        return i;
      }
      return value;
    }).map((_a) => {
      var { value } = _a, rest = __rest(_a, ["value"]);
      return Object.assign(Object.assign({}, rest), { value: protoBase64.enc(value) });
    });
  }
  return o;
}
function errorToJsonBytes(error, jsonWriteOptions) {
  const textEncoder = new TextEncoder();
  try {
    const jsonObject = errorToJson(error, jsonWriteOptions);
    const jsonString = JSON.stringify(jsonObject);
    return textEncoder.encode(jsonString);
  } catch (e) {
    const m = e instanceof Error ? e.message : String(e);
    throw new ConnectError(`failed to serialize Connect Error: ${m}`, Code.Internal);
  }
}

// node_modules/@bufbuild/connect/dist/esm/protocol-connect/end-stream.js
var endStreamFlag = 2;
function endStreamFromJson(data) {
  const parseErr = new ConnectError("invalid end stream", Code.InvalidArgument);
  let jsonValue;
  try {
    jsonValue = JSON.parse(typeof data == "string" ? data : new TextDecoder().decode(data));
  } catch (e) {
    throw parseErr;
  }
  if (typeof jsonValue != "object" || jsonValue == null || Array.isArray(jsonValue)) {
    throw parseErr;
  }
  const metadata = new Headers();
  if ("metadata" in jsonValue) {
    if (typeof jsonValue.metadata != "object" || jsonValue.metadata == null || Array.isArray(jsonValue.metadata)) {
      throw parseErr;
    }
    for (const [key, values] of Object.entries(jsonValue.metadata)) {
      if (!Array.isArray(values) || values.some((value) => typeof value != "string")) {
        throw parseErr;
      }
      for (const value of values) {
        metadata.append(key, value);
      }
    }
  }
  const error = "error" in jsonValue ? errorFromJson(jsonValue.error, metadata, parseErr) : void 0;
  return { metadata, error };
}
function endStreamToJson(metadata, error, jsonWriteOptions) {
  const es = {};
  if (error !== void 0) {
    es.error = errorToJson(error, jsonWriteOptions);
    metadata = appendHeaders(metadata, error.metadata);
  }
  let hasMetadata = false;
  const md = {};
  metadata.forEach((value, key) => {
    hasMetadata = true;
    md[key] = [value];
  });
  if (hasMetadata) {
    es.metadata = md;
  }
  return es;
}
function createEndStreamSerialization(options) {
  const textEncoder = new TextEncoder();
  return {
    serialize(data) {
      try {
        const jsonObject = endStreamToJson(data.metadata, data.error, options);
        const jsonString = JSON.stringify(jsonObject);
        return textEncoder.encode(jsonString);
      } catch (e) {
        const m = e instanceof Error ? e.message : String(e);
        throw new ConnectError(`failed to serialize EndStreamResponse: ${m}`, Code.Internal);
      }
    },
    parse(data) {
      try {
        return endStreamFromJson(data);
      } catch (e) {
        const m = e instanceof Error ? e.message : String(e);
        throw new ConnectError(`failed to parse EndStreamResponse: ${m}`, Code.InvalidArgument);
      }
    }
  };
}

// node_modules/@bufbuild/connect/dist/esm/protocol-connect/headers.js
var headerContentType2 = "Content-Type";
var headerUnaryContentLength = "Content-Length";
var headerUnaryEncoding = "Content-Encoding";
var headerStreamEncoding = "Connect-Content-Encoding";
var headerUnaryAcceptEncoding = "Accept-Encoding";
var headerStreamAcceptEncoding = "Connect-Accept-Encoding";
var headerTimeout2 = "Connect-Timeout-Ms";
var headerProtocolVersion = "Connect-Protocol-Version";

// node_modules/@bufbuild/connect/dist/esm/protocol-connect/http-status.js
function codeFromHttpStatus(httpStatus) {
  switch (httpStatus) {
    case 400:
      return Code.InvalidArgument;
    case 401:
      return Code.Unauthenticated;
    case 403:
      return Code.PermissionDenied;
    case 404:
      return Code.Unimplemented;
    case 408:
      return Code.DeadlineExceeded;
    case 409:
      return Code.Aborted;
    case 412:
      return Code.FailedPrecondition;
    case 413:
      return Code.ResourceExhausted;
    case 415:
      return Code.Internal;
    case 429:
      return Code.Unavailable;
    case 431:
      return Code.ResourceExhausted;
    case 502:
      return Code.Unavailable;
    case 503:
      return Code.Unavailable;
    case 504:
      return Code.Unavailable;
    default:
      return Code.Unknown;
  }
}
function codeToHttpStatus(code) {
  switch (code) {
    case Code.Canceled:
      return 408;
    case Code.Unknown:
      return 500;
    case Code.InvalidArgument:
      return 400;
    case Code.DeadlineExceeded:
      return 408;
    case Code.NotFound:
      return 404;
    case Code.AlreadyExists:
      return 409;
    case Code.PermissionDenied:
      return 403;
    case Code.ResourceExhausted:
      return 429;
    case Code.FailedPrecondition:
      return 412;
    case Code.Aborted:
      return 409;
    case Code.OutOfRange:
      return 400;
    case Code.Unimplemented:
      return 404;
    case Code.Internal:
      return 500;
    case Code.Unavailable:
      return 503;
    case Code.DataLoss:
      return 500;
    case Code.Unauthenticated:
      return 401;
    default:
      return 500;
  }
}

// node_modules/@bufbuild/connect/dist/esm/protocol-connect/parse-timeout.js
function parseTimeout2(value, maxTimeoutMs) {
  if (value === null) {
    return {};
  }
  const results = /^\d{1,10}$/.exec(value);
  if (results === null) {
    return {
      error: new ConnectError(`protocol error: invalid connect timeout value: ${value}`, Code.InvalidArgument)
    };
  }
  const timeoutMs = parseInt(results[0]);
  if (timeoutMs > maxTimeoutMs) {
    return {
      timeoutMs,
      error: new ConnectError(`timeout ${timeoutMs}ms must be <= ${maxTimeoutMs}`, Code.InvalidArgument)
    };
  }
  return {
    timeoutMs: parseInt(results[0])
  };
}

// node_modules/@bufbuild/connect/dist/esm/protocol-connect/query-params.js
var paramConnectVersion = "connect";
var paramEncoding = "encoding";
var paramCompression = "compression";
var paramBase64 = "base64";
var paramMessage = "message";

// node_modules/@bufbuild/connect/dist/esm/protocol-connect/trailer-mux.js
function trailerDemux(header) {
  const h = new Headers(), t = new Headers();
  header.forEach((value, key) => {
    if (key.toLowerCase().startsWith("trailer-")) {
      t.set(key.substring(8), value);
    } else {
      h.set(key, value);
    }
  });
  return [h, t];
}
function trailerMux(header, trailer) {
  const h = new Headers(header);
  trailer.forEach((value, key) => {
    h.set(`trailer-${key}`, value);
  });
  return h;
}

// node_modules/@bufbuild/connect/dist/esm/protocol-connect/version.js
var protocolVersion = "1";
function requireProtocolVersionHeader(requestHeader2) {
  const v = requestHeader2.get(headerProtocolVersion);
  if (v === null) {
    throw new ConnectError(`missing required header: set ${headerProtocolVersion} to "${protocolVersion}"`, Code.InvalidArgument);
  } else if (v !== protocolVersion) {
    throw new ConnectError(`${headerProtocolVersion} must be "${protocolVersion}": got "${v}"`, Code.InvalidArgument);
  }
}
function requireProtocolVersionParam(queryParams) {
  const v = queryParams.get(paramConnectVersion);
  if (v === null) {
    throw new ConnectError(`missing required parameter: set ${paramConnectVersion} to "v${protocolVersion}"`, Code.InvalidArgument);
  } else if (v !== `v${protocolVersion}`) {
    throw new ConnectError(`${paramConnectVersion} must be "v${protocolVersion}": got "${v}"`, Code.InvalidArgument);
  }
}

// node_modules/@bufbuild/connect/dist/esm/protocol-connect/handler-factory.js
var protocolName3 = "connect";
var methodPost3 = "POST";
var methodGet = "GET";
function createHandlerFactory3(options) {
  const opt = validateUniversalHandlerOptions(options);
  const endStreamSerialization = createEndStreamSerialization(opt.jsonOptions);
  function fact(spec) {
    let h;
    let contentTypeRegExp4;
    const serialization = createMethodSerializationLookup(spec.method, opt.binaryOptions, opt.jsonOptions, opt);
    switch (spec.kind) {
      case MethodKind.Unary:
        contentTypeRegExp4 = contentTypeUnaryRegExp;
        h = createUnaryHandler(opt, spec, serialization);
        break;
      default:
        contentTypeRegExp4 = contentTypeStreamRegExp;
        h = createStreamHandler(opt, spec, serialization, endStreamSerialization);
        break;
    }
    const allowedMethods = [methodPost3];
    if (spec.method.idempotency === MethodIdempotency.NoSideEffects) {
      allowedMethods.push(methodGet);
    }
    return Object.assign(h, {
      protocolNames: [protocolName3],
      supportedContentType: contentTypeMatcher(contentTypeRegExp4),
      allowedMethods,
      requestPath: createMethodUrl("/", spec.service, spec.method),
      service: spec.service,
      method: spec.method
    });
  }
  fact.protocolName = protocolName3;
  return fact;
}
function createUnaryHandler(opt, spec, serialization) {
  return async function handle(req) {
    const isGet = req.method == methodGet;
    if (isGet && spec.method.idempotency != MethodIdempotency.NoSideEffects) {
      return uResponseMethodNotAllowed;
    }
    const queryParams = new URL(req.url).searchParams;
    const compressionRequested = isGet ? queryParams.get(paramCompression) : req.header.get(headerUnaryEncoding);
    const type = isGet ? parseEncodingQuery(queryParams.get(paramEncoding)) : parseContentType3(req.header.get(headerContentType2));
    if (type == void 0 || type.stream) {
      return uResponseUnsupportedMediaType;
    }
    const timeout = parseTimeout2(req.header.get(headerTimeout2), opt.maxTimeoutMs);
    const context = createHandlerContext(Object.assign(Object.assign({}, spec), { requestMethod: req.method, protocolName: protocolName3, timeoutMs: timeout.timeoutMs, shutdownSignal: opt.shutdownSignal, requestSignal: req.signal, requestHeader: req.header, responseHeader: {
      [headerContentType2]: type.binary ? contentTypeUnaryProto : contentTypeUnaryJson
    } }));
    const compression = compressionNegotiate(opt.acceptCompression, compressionRequested, req.header.get(headerUnaryAcceptEncoding), headerUnaryAcceptEncoding);
    let status = uResponseOk.status;
    let body;
    try {
      if (opt.requireConnectProtocolHeader) {
        if (isGet) {
          requireProtocolVersionParam(queryParams);
        } else {
          requireProtocolVersionHeader(req.header);
        }
      }
      if (compression.error) {
        throw compression.error;
      }
      if (timeout.error) {
        throw timeout.error;
      }
      let reqBody;
      if (isGet) {
        reqBody = await readUnaryMessageFromQuery(opt.readMaxBytes, compression.request, queryParams);
      } else {
        reqBody = await readUnaryMessageFromBody(opt.readMaxBytes, compression.request, req);
      }
      const input = parseUnaryMessage(spec.method, type.binary, serialization, reqBody);
      const output = await invokeUnaryImplementation(spec, context, input);
      body = serialization.getO(type.binary).serialize(output);
    } catch (e) {
      let error;
      if (e instanceof ConnectError) {
        error = e;
      } else {
        error = new ConnectError("internal error", Code.Internal, void 0, void 0, e);
      }
      status = codeToHttpStatus(error.code);
      context.responseHeader.set(headerContentType2, contentTypeUnaryJson);
      error.metadata.forEach((value, key) => {
        context.responseHeader.set(key, value);
      });
      body = errorToJsonBytes(error, opt.jsonOptions);
    } finally {
      context.abort();
    }
    if (compression.response && body.byteLength >= opt.compressMinBytes) {
      body = await compression.response.compress(body);
      context.responseHeader.set(headerUnaryEncoding, compression.response.name);
    }
    const header = trailerMux(context.responseHeader, context.responseTrailer);
    header.set(headerUnaryContentLength, body.byteLength.toString(10));
    return {
      status,
      body: createAsyncIterable([body]),
      header
    };
  };
}
async function readUnaryMessageFromBody(readMaxBytes, compression, request) {
  if (typeof request.body == "object" && request.body !== null && Symbol.asyncIterator in request.body) {
    let reqBytes = await readAllBytes(request.body, readMaxBytes, request.header.get(headerUnaryContentLength));
    if (compression) {
      reqBytes = await compression.decompress(reqBytes, readMaxBytes);
    }
    return reqBytes;
  }
  return request.body;
}
async function readUnaryMessageFromQuery(readMaxBytes, compression, queryParams) {
  var _a;
  const base64 = queryParams.get(paramBase64);
  const message = (_a = queryParams.get(paramMessage)) !== null && _a !== void 0 ? _a : "";
  let decoded;
  if (base64 === "1") {
    decoded = protoBase64.dec(message);
  } else {
    decoded = new TextEncoder().encode(message);
  }
  if (compression) {
    decoded = await compression.decompress(decoded, readMaxBytes);
  }
  return decoded;
}
function parseUnaryMessage(method, useBinaryFormat, serialization, input) {
  if (input instanceof Uint8Array) {
    return serialization.getI(useBinaryFormat).parse(input);
  }
  if (useBinaryFormat) {
    throw new ConnectError("received parsed JSON request body, but content-type indicates binary format", Code.Internal);
  }
  try {
    return method.I.fromJson(input);
  } catch (e) {
    throw ConnectError.from(e, Code.InvalidArgument);
  }
}
function createStreamHandler(opt, spec, serialization, endStreamSerialization) {
  return async function handle(req) {
    assertByteStreamRequest(req);
    const type = parseContentType3(req.header.get(headerContentType2));
    if (type == void 0 || !type.stream) {
      return uResponseUnsupportedMediaType;
    }
    if (req.method !== methodPost3) {
      return uResponseMethodNotAllowed;
    }
    const timeout = parseTimeout2(req.header.get(headerTimeout2), opt.maxTimeoutMs);
    const context = createHandlerContext(Object.assign(Object.assign({}, spec), { requestMethod: req.method, protocolName: protocolName3, timeoutMs: timeout.timeoutMs, shutdownSignal: opt.shutdownSignal, requestSignal: req.signal, requestHeader: req.header, responseHeader: {
      [headerContentType2]: type.binary ? contentTypeStreamProto : contentTypeStreamJson
    } }));
    const compression = compressionNegotiate(opt.acceptCompression, req.header.get(headerStreamEncoding), req.header.get(headerStreamAcceptEncoding), headerStreamAcceptEncoding);
    if (compression.response) {
      context.responseHeader.set(headerStreamEncoding, compression.response.name);
    }
    const outputIt = pipe(req.body, transformPrepend(() => {
      if (opt.requireConnectProtocolHeader) {
        requireProtocolVersionHeader(req.header);
      }
      if (compression.error)
        throw compression.error;
      if (timeout.error)
        throw timeout.error;
      return void 0;
    }), transformSplitEnvelope(opt.readMaxBytes), transformDecompressEnvelope(compression.request, opt.readMaxBytes), transformParseEnvelope(serialization.getI(type.binary), endStreamFlag), transformInvokeImplementation(spec, context), transformSerializeEnvelope(serialization.getO(type.binary)), transformCatchFinally((e) => {
      context.abort();
      const end = {
        metadata: context.responseTrailer
      };
      if (e instanceof ConnectError) {
        end.error = e;
      } else if (e !== void 0) {
        end.error = new ConnectError("internal error", Code.Internal, void 0, void 0, e);
      }
      return {
        flags: endStreamFlag,
        data: endStreamSerialization.serialize(end)
      };
    }), transformCompressEnvelope(compression.response, opt.compressMinBytes), transformJoinEnvelopes());
    return Object.assign(Object.assign({}, uResponseOk), {
      // We wait for the first response body bytes before resolving, so that
      // implementations have a chance to add headers before an adapter commits
      // them to the wire.
      body: await untilFirst(outputIt),
      header: context.responseHeader
    });
  };
}

// node_modules/@bufbuild/connect/dist/esm/router.js
function createConnectRouter(routerOptions) {
  const base = whichProtocols(routerOptions);
  const handlers = [];
  return {
    handlers,
    service(service, implementation, options) {
      const { protocols } = whichProtocols(options, base);
      handlers.push(...createUniversalServiceHandlers(createServiceImplSpec(service, implementation), protocols));
      return this;
    },
    rpc(service, method, implementation, options) {
      const { protocols } = whichProtocols(options, base);
      handlers.push(createUniversalMethodHandler(createMethodImplSpec(service, method, implementation), protocols));
      return this;
    }
  };
}
function whichProtocols(options, base) {
  if (base && !options) {
    return base;
  }
  const opt = base ? Object.assign(Object.assign({}, validateUniversalHandlerOptions(base.options)), options) : Object.assign(Object.assign({}, options), validateUniversalHandlerOptions(options !== null && options !== void 0 ? options : {}));
  const protocols = [];
  if ((options === null || options === void 0 ? void 0 : options.grpc) !== false) {
    protocols.push(createHandlerFactory2(opt));
  }
  if ((options === null || options === void 0 ? void 0 : options.grpcWeb) !== false) {
    protocols.push(createHandlerFactory(opt));
  }
  if ((options === null || options === void 0 ? void 0 : options.connect) !== false) {
    protocols.push(createHandlerFactory3(opt));
  }
  if (protocols.length === 0) {
    throw new ConnectError("cannot create handler, all protocols are disabled", Code.InvalidArgument);
  }
  return {
    options: opt,
    protocols
  };
}

// node_modules/@bufbuild/connect/dist/esm/cors.js
var cors = {
  /**
   * Request methods that scripts running in the browser are permitted to use.
   *
   * To support cross-domain requests with the protocols supported by Connect,
   * these headers fields must be included in the preflight response header
   * Access-Control-Allow-Methods.
   */
  allowedMethods: ["POST", "GET"],
  /**
   * Header fields that scripts running in the browser are permitted to send.
   *
   * To support cross-domain requests with the protocols supported by Connect,
   * these field names must be included in the preflight response header
   * Access-Control-Allow-Headers.
   *
   * Make sure to include any application-specific headers your browser client
   * may send.
   */
  allowedHeaders: [
    headerContentType2,
    headerProtocolVersion,
    headerTimeout2,
    headerStreamEncoding,
    headerStreamAcceptEncoding,
    headerUnaryEncoding,
    headerUnaryAcceptEncoding,
    headerMessageType,
    headerXGrpcWeb,
    headerXUserAgent,
    headerTimeout
  ],
  /**
   * Header fields that scripts running the browser are permitted to see.
   *
   * To support cross-domain requests with the protocols supported by Connect,
   * these field names must be included in header Access-Control-Expose-Headers
   * of the actual response.
   *
   * Make sure to include any application-specific headers your browser client
   * should see. If your application uses trailers, they will be sent as header
   * fields with a `Trailer-` prefix for Connect unary RPCs - make sure to
   * expose them as well if you want them to be visible in all supported
   * protocols.
   */
  exposedHeaders: [
    headerGrpcStatus,
    headerGrpcMessage,
    headerStatusDetailsBin,
    headerUnaryEncoding,
    headerStreamEncoding
    // Unused in web browsers, but added for future-proofing
  ]
};

// node_modules/@bufbuild/connect/dist/esm/legacy-interceptor.js
function runUnary(req, next, interceptors) {
  if (interceptors) {
    next = applyInterceptors(next, interceptors);
  }
  return next(req);
}
function runStreaming(req, next, interceptors) {
  if (interceptors) {
    next = applyInterceptors(next, interceptors);
  }
  return next(req);
}
function applyInterceptors(next, interceptors) {
  return interceptors.concat().reverse().reduce(
    // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
    (n, i) => i(n),
    next
  );
}

// node_modules/@bufbuild/connect/dist/esm/protocol-connect/request-header.js
function requestHeader(methodKind, useBinaryFormat, timeoutMs, userProvidedHeaders) {
  const result = new Headers(userProvidedHeaders !== null && userProvidedHeaders !== void 0 ? userProvidedHeaders : {});
  if (timeoutMs !== void 0) {
    result.set(headerTimeout2, `${timeoutMs}`);
  }
  result.set(headerContentType2, methodKind == MethodKind.Unary ? useBinaryFormat ? contentTypeUnaryProto : contentTypeUnaryJson : useBinaryFormat ? contentTypeStreamProto : contentTypeStreamJson);
  result.set(headerProtocolVersion, protocolVersion);
  return result;
}
function requestHeaderWithCompression(methodKind, useBinaryFormat, timeoutMs, userProvidedHeaders, acceptCompression, sendCompression) {
  const result = requestHeader(methodKind, useBinaryFormat, timeoutMs, userProvidedHeaders);
  if (sendCompression != null) {
    const name = methodKind == MethodKind.Unary ? headerUnaryEncoding : headerStreamEncoding;
    result.set(name, sendCompression.name);
  }
  if (acceptCompression.length > 0) {
    const name = methodKind == MethodKind.Unary ? headerUnaryAcceptEncoding : headerStreamAcceptEncoding;
    result.set(name, acceptCompression.map((c) => c.name).join(","));
  }
  return result;
}

// node_modules/@bufbuild/connect/dist/esm/protocol-connect/validate-response.js
function validateResponse(methodKind, status, headers) {
  const mimeType = headers.get("Content-Type");
  const parsedType = parseContentType3(mimeType);
  if (status !== 200) {
    const errorFromStatus = new ConnectError(`HTTP ${status}`, codeFromHttpStatus(status), headers);
    if (methodKind == MethodKind.Unary && parsedType && !parsedType.binary) {
      return { isUnaryError: true, unaryError: errorFromStatus };
    }
    throw errorFromStatus;
  }
  return { isUnaryError: false };
}
function validateResponseWithCompression(methodKind, acceptCompression, status, headers) {
  let compression;
  const encoding = headers.get(methodKind == MethodKind.Unary ? headerUnaryEncoding : headerStreamEncoding);
  if (encoding != null && encoding.toLowerCase() !== "identity") {
    compression = acceptCompression.find((c) => c.name === encoding);
    if (!compression) {
      throw new ConnectError(`unsupported response encoding "${encoding}"`, Code.InvalidArgument, headers);
    }
  }
  return Object.assign({ compression }, validateResponse(methodKind, status, headers));
}

// node_modules/@bufbuild/connect/dist/esm/protocol-connect/get-request.js
var contentTypePrefix = "application/";
function encodeMessageForUrl(message, useBase64) {
  if (useBase64) {
    return protoBase64.enc(message).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
  } else {
    return encodeURIComponent(new TextDecoder().decode(message));
  }
}
function transformConnectPostToGetRequest(request, message, useBase64) {
  let query = `?connect=v${protocolVersion}`;
  const contentType = request.header.get(headerContentType2);
  if ((contentType === null || contentType === void 0 ? void 0 : contentType.indexOf(contentTypePrefix)) === 0) {
    query += "&encoding=" + encodeURIComponent(contentType.slice(contentTypePrefix.length));
  }
  const compression = request.header.get(headerUnaryEncoding);
  if (compression !== null && compression !== "identity") {
    query += "&compression=" + encodeURIComponent(compression);
    useBase64 = true;
  }
  if (useBase64) {
    query += "&base64=1";
  }
  query += "&message=" + encodeMessageForUrl(message, useBase64);
  const url = request.url + query;
  const header = new Headers(request.header);
  header.delete(headerProtocolVersion);
  header.delete(headerContentType2);
  header.delete(headerUnaryContentLength);
  header.delete(headerUnaryEncoding);
  header.delete(headerUnaryAcceptEncoding);
  return Object.assign(Object.assign({}, request), {
    init: Object.assign(Object.assign({}, request.init), { method: "GET" }),
    url,
    header
  });
}

// node_modules/@bufbuild/connect/dist/esm/protocol/run-call.js
function runUnaryCall(opt) {
  const next = applyInterceptors2(opt.next, opt.interceptors);
  const [signal, abort, done] = setupSignal(opt);
  const req = Object.assign(Object.assign({}, opt.req), { message: normalize(opt.req.method.I, opt.req.message), signal });
  return next(req).then((res) => {
    done();
    return res;
  }, abort);
}
function runStreamingCall(opt) {
  const next = applyInterceptors2(opt.next, opt.interceptors);
  const [signal, abort, done] = setupSignal(opt);
  const req = Object.assign(Object.assign({}, opt.req), { message: normalizeIterable(opt.req.method.I, opt.req.message), signal });
  let doneCalled = false;
  signal.addEventListener("abort", function() {
    var _a, _b;
    const it = opt.req.message[Symbol.asyncIterator]();
    if (!doneCalled) {
      (_a = it.throw) === null || _a === void 0 ? void 0 : _a.call(it, this.reason).catch(() => {
      });
    }
    (_b = it.return) === null || _b === void 0 ? void 0 : _b.call(it).catch(() => {
    });
  });
  return next(req).then((res) => {
    return Object.assign(Object.assign({}, res), { message: {
      [Symbol.asyncIterator]() {
        const it = res.message[Symbol.asyncIterator]();
        return {
          next() {
            return it.next().then((r) => {
              if (r.done == true) {
                doneCalled = true;
                done();
              }
              return r;
            }, abort);
          }
          // We deliberately omit throw/return.
        };
      }
    } });
  }, abort);
}
function setupSignal(opt) {
  const { signal, cleanup } = createDeadlineSignal(opt.timeoutMs);
  const controller = createLinkedAbortController(opt.signal, signal);
  return [
    controller.signal,
    function abort(reason) {
      const e = ConnectError.from(signal.aborted ? getAbortSignalReason(signal) : reason);
      controller.abort(e);
      cleanup();
      return Promise.reject(e);
    },
    function done() {
      cleanup();
      controller.abort();
    }
  ];
}
function applyInterceptors2(next, interceptors) {
  var _a;
  return (_a = interceptors === null || interceptors === void 0 ? void 0 : interceptors.concat().reverse().reduce(
    // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
    (n, i) => i(n),
    next
  )) !== null && _a !== void 0 ? _a : next;
}
function normalize(type, message) {
  return message instanceof type ? message : new type(message);
}
function normalizeIterable(messageType, input) {
  function transform(result) {
    if (result.done === true) {
      return result;
    }
    return {
      done: result.done,
      value: normalize(messageType, result.value)
    };
  }
  return {
    [Symbol.asyncIterator]() {
      const it = input[Symbol.asyncIterator]();
      const res = {
        next: () => it.next().then(transform)
      };
      if (it.throw !== void 0) {
        res.throw = (e) => it.throw(e).then(transform);
      }
      if (it.return !== void 0) {
        res.return = (v) => it.return(v).then(transform);
      }
      return res;
    }
  };
}

// node_modules/@bufbuild/connect/dist/esm/protocol-connect/transport.js
var __asyncValues5 = function(o) {
  if (!Symbol.asyncIterator)
    throw new TypeError("Symbol.asyncIterator is not defined.");
  var m = o[Symbol.asyncIterator], i;
  return m ? m.call(o) : (o = typeof __values === "function" ? __values(o) : o[Symbol.iterator](), i = {}, verb("next"), verb("throw"), verb("return"), i[Symbol.asyncIterator] = function() {
    return this;
  }, i);
  function verb(n) {
    i[n] = o[n] && function(v) {
      return new Promise(function(resolve, reject) {
        v = o[n](v), settle(resolve, reject, v.done, v.value);
      });
    };
  }
  function settle(resolve, reject, d, v) {
    Promise.resolve(v).then(function(v2) {
      resolve({ value: v2, done: d });
    }, reject);
  }
};
var __await4 = function(v) {
  return this instanceof __await4 ? (this.v = v, this) : new __await4(v);
};
var __asyncGenerator4 = function(thisArg, _arguments, generator) {
  if (!Symbol.asyncIterator)
    throw new TypeError("Symbol.asyncIterator is not defined.");
  var g = generator.apply(thisArg, _arguments || []), i, q = [];
  return i = {}, verb("next"), verb("throw"), verb("return"), i[Symbol.asyncIterator] = function() {
    return this;
  }, i;
  function verb(n) {
    if (g[n])
      i[n] = function(v) {
        return new Promise(function(a, b) {
          q.push([n, v, a, b]) > 1 || resume(n, v);
        });
      };
  }
  function resume(n, v) {
    try {
      step(g[n](v));
    } catch (e) {
      settle(q[0][3], e);
    }
  }
  function step(r) {
    r.value instanceof __await4 ? Promise.resolve(r.value.v).then(fulfill, reject) : settle(q[0][2], r);
  }
  function fulfill(value) {
    resume("next", value);
  }
  function reject(value) {
    resume("throw", value);
  }
  function settle(f, v) {
    if (f(v), q.shift(), q.length)
      resume(q[0][0], q[0][1]);
  }
};
function createTransport(opt) {
  return {
    async unary(service, method, signal, timeoutMs, header, message) {
      const serialization = createMethodSerializationLookup(method, opt.binaryOptions, opt.jsonOptions, opt);
      return await runUnaryCall({
        interceptors: opt.interceptors,
        signal,
        timeoutMs,
        req: {
          stream: false,
          service,
          method,
          url: createMethodUrl(opt.baseUrl, service, method),
          init: {},
          header: requestHeaderWithCompression(method.kind, opt.useBinaryFormat, timeoutMs, header, opt.acceptCompression, opt.sendCompression),
          message
        },
        next: async (req) => {
          var _a;
          let requestBody = serialization.getI(opt.useBinaryFormat).serialize(req.message);
          if (opt.sendCompression && requestBody.byteLength > opt.compressMinBytes) {
            requestBody = await opt.sendCompression.compress(requestBody);
            req.header.set(headerUnaryEncoding, opt.sendCompression.name);
          } else {
            req.header.delete(headerUnaryEncoding);
          }
          const useGet = opt.useHttpGet === true && method.idempotency === MethodIdempotency.NoSideEffects;
          let body;
          if (useGet) {
            req = transformConnectPostToGetRequest(req, requestBody, opt.useBinaryFormat);
          } else {
            body = createAsyncIterable([requestBody]);
          }
          const universalResponse = await opt.httpClient({
            url: req.url,
            method: (_a = req.init.method) !== null && _a !== void 0 ? _a : "POST",
            header: req.header,
            signal: req.signal,
            body
          });
          const { compression, isUnaryError, unaryError } = validateResponseWithCompression(method.kind, opt.acceptCompression, universalResponse.status, universalResponse.header);
          const [header2, trailer] = trailerDemux(universalResponse.header);
          let responseBody = await pipeTo(universalResponse.body, sinkAllBytes(opt.readMaxBytes, universalResponse.header.get(headerUnaryContentLength)), { propagateDownStreamError: false });
          if (compression) {
            responseBody = await compression.decompress(responseBody, opt.readMaxBytes);
          }
          if (isUnaryError) {
            throw errorFromJsonBytes(responseBody, appendHeaders(header2, trailer), unaryError);
          }
          return {
            stream: false,
            service,
            method,
            header: header2,
            message: serialization.getO(opt.useBinaryFormat).parse(responseBody),
            trailer
          };
        }
      });
    },
    async stream(service, method, signal, timeoutMs, header, input) {
      const serialization = createMethodSerializationLookup(method, opt.binaryOptions, opt.jsonOptions, opt);
      const endStreamSerialization = createEndStreamSerialization(opt.jsonOptions);
      return runStreamingCall({
        interceptors: opt.interceptors,
        signal,
        timeoutMs,
        req: {
          stream: true,
          service,
          method,
          url: createMethodUrl(opt.baseUrl, service, method),
          init: {
            method: "POST",
            redirect: "error",
            mode: "cors"
          },
          header: requestHeaderWithCompression(method.kind, opt.useBinaryFormat, timeoutMs, header, opt.acceptCompression, opt.sendCompression),
          message: input
        },
        next: async (req) => {
          const uRes = await opt.httpClient({
            url: req.url,
            method: "POST",
            header: req.header,
            signal: req.signal,
            body: pipe(req.message, transformSerializeEnvelope(serialization.getI(opt.useBinaryFormat)), transformCompressEnvelope(opt.sendCompression, opt.compressMinBytes), transformJoinEnvelopes(), { propagateDownStreamError: true })
          });
          const { compression } = validateResponseWithCompression(method.kind, opt.acceptCompression, uRes.status, uRes.header);
          const res = Object.assign(Object.assign({}, req), { header: uRes.header, trailer: new Headers(), message: pipe(uRes.body, transformSplitEnvelope(opt.readMaxBytes), transformDecompressEnvelope(compression !== null && compression !== void 0 ? compression : null, opt.readMaxBytes), transformParseEnvelope(serialization.getO(opt.useBinaryFormat), endStreamFlag, endStreamSerialization), function(iterable) {
            return __asyncGenerator4(this, arguments, function* () {
              var _a, e_1, _b, _c;
              let endStreamReceived = false;
              try {
                for (var _d = true, iterable_1 = __asyncValues5(iterable), iterable_1_1; iterable_1_1 = yield __await4(iterable_1.next()), _a = iterable_1_1.done, !_a; _d = true) {
                  _c = iterable_1_1.value;
                  _d = false;
                  const chunk = _c;
                  if (chunk.end) {
                    if (endStreamReceived) {
                      throw new ConnectError("protocol error: received extra EndStreamResponse", Code.InvalidArgument);
                    }
                    endStreamReceived = true;
                    if (chunk.value.error) {
                      throw chunk.value.error;
                    }
                    chunk.value.metadata.forEach((value, key) => res.trailer.set(key, value));
                    continue;
                  }
                  if (endStreamReceived) {
                    throw new ConnectError("protocol error: received extra message after EndStreamResponse", Code.InvalidArgument);
                  }
                  yield yield __await4(chunk.value);
                }
              } catch (e_1_1) {
                e_1 = { error: e_1_1 };
              } finally {
                try {
                  if (!_d && !_a && (_b = iterable_1.return))
                    yield __await4(_b.call(iterable_1));
                } finally {
                  if (e_1)
                    throw e_1.error;
                }
              }
              if (!endStreamReceived) {
                throw new ConnectError("protocol error: missing EndStreamResponse", Code.InvalidArgument);
              }
            });
          }, { propagateDownStreamError: true }) });
          return res;
        }
      });
    }
  };
}

// node_modules/@bufbuild/connect/dist/esm/protocol/universal-handler-client.js
function createUniversalHandlerClient(uHandlers) {
  const handlerMap = /* @__PURE__ */ new Map();
  for (const handler of uHandlers) {
    handlerMap.set(handler.requestPath, handler);
  }
  return async (uClientReq) => {
    var _a, _b, _c;
    const pathname = new URL(uClientReq.url).pathname;
    const handler = handlerMap.get(pathname);
    if (!handler) {
      throw new ConnectError(`RouterHttpClient: no handler registered for ${pathname}`, Code.Unimplemented);
    }
    const reqSignal = (_a = uClientReq.signal) !== null && _a !== void 0 ? _a : new AbortController().signal;
    const uServerRes = await raceSignal(reqSignal, handler({
      body: (_b = uClientReq.body) !== null && _b !== void 0 ? _b : createAsyncIterable([]),
      httpVersion: "2.0",
      method: uClientReq.method,
      url: uClientReq.url,
      header: uClientReq.header,
      signal: reqSignal
    }));
    const body = (_c = uServerRes.body) !== null && _c !== void 0 ? _c : createAsyncIterable([]);
    return {
      body: pipe(body, (iterable) => {
        return {
          [Symbol.asyncIterator]() {
            const it = iterable[Symbol.asyncIterator]();
            const w = {
              next() {
                return raceSignal(reqSignal, it.next());
              }
            };
            if (it.throw !== void 0) {
              w.throw = (e) => it.throw(e);
            }
            if (it.return !== void 0) {
              w.return = (value) => it.return(value);
            }
            return w;
          }
        };
      }),
      header: new Headers(uServerRes.header),
      status: uServerRes.status,
      trailer: new Headers(uServerRes.trailer)
    };
  };
}
function raceSignal(signal, promise) {
  let cleanup;
  const signalPromise = new Promise((_, reject) => {
    const onAbort = () => reject(getAbortSignalReason(signal));
    if (signal.aborted) {
      return onAbort();
    }
    signal.addEventListener("abort", onAbort);
    cleanup = () => signal.removeEventListener("abort", onAbort);
  });
  return Promise.race([signalPromise, promise]).finally(cleanup);
}

// node_modules/@bufbuild/connect/dist/esm/router-transport.js
function createRouterTransport(routes, options) {
  var _a, _b;
  const router = createConnectRouter(Object.assign(Object.assign({}, (_a = options === null || options === void 0 ? void 0 : options.router) !== null && _a !== void 0 ? _a : {}), { connect: true }));
  routes(router);
  return createTransport(Object.assign({ httpClient: createUniversalHandlerClient(router.handlers), baseUrl: "https://in-memory", useBinaryFormat: true, interceptors: [], acceptCompression: [], sendCompression: null, compressMinBytes: Number.MAX_SAFE_INTEGER, readMaxBytes: Number.MAX_SAFE_INTEGER, writeMaxBytes: Number.MAX_SAFE_INTEGER }, (_b = options === null || options === void 0 ? void 0 : options.transport) !== null && _b !== void 0 ? _b : {}));
}

export {
  Code,
  ConnectError,
  connectErrorDetails,
  connectErrorFromReason,
  encodeBinaryHeader,
  decodeBinaryHeader,
  appendHeaders,
  makeAnyClient,
  createEnvelopeReadableStream,
  encodeEnvelope,
  createCallbackClient,
  createPromiseClient,
  createHandlerContext,
  createMethodImplSpec,
  createServiceImplSpec,
  trailerFlag,
  trailerParse,
  headerContentType,
  headerTimeout,
  headerGrpcStatus,
  headerGrpcMessage,
  headerXUserAgent,
  headerXGrpcWeb,
  contentTypeProto,
  contentTypeJson,
  findTrailerError,
  createMethodUrl,
  getJsonOptions,
  createClientMethodSerializers,
  errorFromJson,
  endStreamFlag,
  endStreamFromJson,
  trailerDemux,
  createConnectRouter,
  cors,
  runUnary,
  runStreaming,
  requestHeader,
  validateResponse,
  transformConnectPostToGetRequest,
  runUnaryCall,
  runStreamingCall,
  createRouterTransport
};
//# sourceMappingURL=chunk-J5P5UBGQ.js.map
