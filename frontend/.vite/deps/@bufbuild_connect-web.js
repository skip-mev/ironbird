import {
  Code,
  ConnectError,
  appendHeaders,
  contentTypeJson,
  contentTypeProto,
  createClientMethodSerializers,
  createEnvelopeReadableStream,
  createMethodUrl,
  encodeEnvelope,
  endStreamFlag,
  endStreamFromJson,
  errorFromJson,
  findTrailerError,
  getJsonOptions,
  headerContentType,
  headerGrpcMessage,
  headerGrpcStatus,
  headerTimeout,
  headerXGrpcWeb,
  headerXUserAgent,
  requestHeader,
  runStreamingCall,
  runUnaryCall,
  trailerDemux,
  trailerFlag,
  trailerParse,
  transformConnectPostToGetRequest,
  validateResponse
} from "./chunk-J5P5UBGQ.js";
import {
  MethodIdempotency,
  MethodKind
} from "./chunk-TNC6V2Y3.js";
import "./chunk-TITDT5VP.js";

// node_modules/@bufbuild/connect-web/dist/esm/assert-fetch-api.js
function assertFetchApi() {
  try {
    new Headers();
  } catch (_) {
    throw new Error("connect-web requires the fetch API. Are you running on an old version of Node.js? Node.js is not supported in Connect for Web - please stay tuned for Connect for Node.");
  }
}

// node_modules/@bufbuild/connect-web/dist/esm/connect-transport.js
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
function createConnectTransport(options) {
  var _a;
  assertFetchApi();
  const useBinaryFormat = (_a = options.useBinaryFormat) !== null && _a !== void 0 ? _a : false;
  return {
    async unary(service, method, signal, timeoutMs, header, message) {
      var _a2;
      const { serialize, parse } = createClientMethodSerializers(method, useBinaryFormat, options.jsonOptions, options.binaryOptions);
      return await runUnaryCall({
        interceptors: options.interceptors,
        signal,
        timeoutMs,
        req: {
          stream: false,
          service,
          method,
          url: createMethodUrl(options.baseUrl, service, method),
          init: {
            method: "POST",
            credentials: (_a2 = options.credentials) !== null && _a2 !== void 0 ? _a2 : "same-origin",
            redirect: "error",
            mode: "cors"
          },
          header: requestHeader(method.kind, useBinaryFormat, timeoutMs, header),
          message
        },
        next: async (req) => {
          var _a3;
          const useGet = options.useHttpGet === true && method.idempotency === MethodIdempotency.NoSideEffects;
          let body = null;
          if (useGet) {
            req = transformConnectPostToGetRequest(req, serialize(req.message), useBinaryFormat);
          } else {
            body = serialize(req.message);
          }
          const fetch = (_a3 = options.fetch) !== null && _a3 !== void 0 ? _a3 : globalThis.fetch;
          const response = await fetch(req.url, Object.assign(Object.assign({}, req.init), { headers: req.header, signal: req.signal, body }));
          const { isUnaryError, unaryError } = validateResponse(method.kind, response.status, response.headers);
          if (isUnaryError) {
            throw errorFromJson(await response.json(), appendHeaders(...trailerDemux(response.headers)), unaryError);
          }
          const [demuxedHeader, demuxedTrailer] = trailerDemux(response.headers);
          return {
            stream: false,
            service,
            method,
            header: demuxedHeader,
            message: useBinaryFormat ? parse(new Uint8Array(await response.arrayBuffer())) : method.O.fromJson(await response.json(), getJsonOptions(options.jsonOptions)),
            trailer: demuxedTrailer
          };
        }
      });
    },
    async stream(service, method, signal, timeoutMs, header, input) {
      var _a2;
      const { serialize, parse } = createClientMethodSerializers(method, useBinaryFormat, options.jsonOptions, options.binaryOptions);
      function parseResponseBody(body, trailerTarget) {
        return __asyncGenerator(this, arguments, function* parseResponseBody_1() {
          const reader = createEnvelopeReadableStream(body).getReader();
          let endStreamReceived = false;
          for (; ; ) {
            const result = yield __await(reader.read());
            if (result.done) {
              break;
            }
            const { flags, data } = result.value;
            if ((flags & endStreamFlag) === endStreamFlag) {
              endStreamReceived = true;
              const endStream = endStreamFromJson(data);
              if (endStream.error) {
                throw endStream.error;
              }
              endStream.metadata.forEach((value, key) => trailerTarget.set(key, value));
              continue;
            }
            yield yield __await(parse(data));
          }
          if (!endStreamReceived) {
            throw "missing EndStreamResponse";
          }
        });
      }
      async function createRequestBody(input2) {
        if (method.kind != MethodKind.ServerStreaming) {
          throw "The fetch API does not support streaming request bodies";
        }
        const r = await input2[Symbol.asyncIterator]().next();
        if (r.done == true) {
          throw "missing request message";
        }
        return encodeEnvelope(0, serialize(r.value));
      }
      return await runStreamingCall({
        interceptors: options.interceptors,
        timeoutMs,
        signal,
        req: {
          stream: true,
          service,
          method,
          url: createMethodUrl(options.baseUrl, service, method),
          init: {
            method: "POST",
            credentials: (_a2 = options.credentials) !== null && _a2 !== void 0 ? _a2 : "same-origin",
            redirect: "error",
            mode: "cors"
          },
          header: requestHeader(method.kind, useBinaryFormat, timeoutMs, header),
          message: input
        },
        next: async (req) => {
          var _a3;
          const fetch = (_a3 = options.fetch) !== null && _a3 !== void 0 ? _a3 : globalThis.fetch;
          const fRes = await fetch(req.url, Object.assign(Object.assign({}, req.init), { headers: req.header, signal: req.signal, body: await createRequestBody(req.message) }));
          validateResponse(method.kind, fRes.status, fRes.headers);
          if (fRes.body === null) {
            throw "missing response body";
          }
          const trailer = new Headers();
          const res = Object.assign(Object.assign({}, req), { header: fRes.headers, trailer, message: parseResponseBody(fRes.body, trailer) });
          return res;
        }
      });
    }
  };
}

// node_modules/@bufbuild/connect/dist/esm/protocol-grpc/validate-trailer.js
function validateTrailer(trailer) {
  const err = findTrailerError(trailer);
  if (err) {
    throw err;
  }
}

// node_modules/@bufbuild/connect/dist/esm/protocol-grpc-web/request-header.js
function requestHeader2(useBinaryFormat, timeoutMs, userProvidedHeaders) {
  const result = new Headers(userProvidedHeaders !== null && userProvidedHeaders !== void 0 ? userProvidedHeaders : {});
  result.set(headerContentType, useBinaryFormat ? contentTypeProto : contentTypeJson);
  result.set(headerXGrpcWeb, "1");
  result.set(headerXUserAgent, "connect-es/0.13.0");
  if (timeoutMs !== void 0) {
    result.set(headerTimeout, `${timeoutMs}m`);
  }
  return result;
}

// node_modules/@bufbuild/connect/dist/esm/protocol-grpc/http-status.js
function codeFromHttpStatus2(httpStatus) {
  switch (httpStatus) {
    case 400:
      return Code.Internal;
    case 401:
      return Code.Unauthenticated;
    case 403:
      return Code.PermissionDenied;
    case 404:
      return Code.Unimplemented;
    case 429:
      return Code.Unavailable;
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

// node_modules/@bufbuild/connect/dist/esm/protocol-grpc-web/validate-response.js
function validateResponse2(status, headers) {
  var _a;
  if (status >= 200 && status < 300) {
    const err = findTrailerError(headers);
    if (err) {
      throw err;
    }
    return { foundStatus: headers.has(headerGrpcStatus) };
  }
  throw new ConnectError(decodeURIComponent((_a = headers.get(headerGrpcMessage)) !== null && _a !== void 0 ? _a : `HTTP ${status}`), codeFromHttpStatus2(status), headers);
}

// node_modules/@bufbuild/connect-web/dist/esm/grpc-web-transport.js
var __await2 = function(v) {
  return this instanceof __await2 ? (this.v = v, this) : new __await2(v);
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
function createGrpcWebTransport(options) {
  var _a;
  assertFetchApi();
  const useBinaryFormat = (_a = options.useBinaryFormat) !== null && _a !== void 0 ? _a : true;
  return {
    async unary(service, method, signal, timeoutMs, header, message) {
      var _a2;
      const { serialize, parse } = createClientMethodSerializers(method, useBinaryFormat, options.jsonOptions, options.binaryOptions);
      return await runUnaryCall({
        interceptors: options.interceptors,
        signal,
        timeoutMs,
        req: {
          stream: false,
          service,
          method,
          url: createMethodUrl(options.baseUrl, service, method),
          init: {
            method: "POST",
            credentials: (_a2 = options.credentials) !== null && _a2 !== void 0 ? _a2 : "same-origin",
            redirect: "error",
            mode: "cors"
          },
          header: requestHeader2(useBinaryFormat, timeoutMs, header),
          message
        },
        next: async (req) => {
          var _a3;
          const fetch = (_a3 = options.fetch) !== null && _a3 !== void 0 ? _a3 : globalThis.fetch;
          const response = await fetch(req.url, Object.assign(Object.assign({}, req.init), { headers: req.header, signal: req.signal, body: encodeEnvelope(0, serialize(req.message)) }));
          validateResponse2(response.status, response.headers);
          if (!response.body) {
            throw "missing response body";
          }
          const reader = createEnvelopeReadableStream(response.body).getReader();
          let trailer;
          let message2;
          for (; ; ) {
            const r = await reader.read();
            if (r.done) {
              break;
            }
            const { flags, data } = r.value;
            if (flags === trailerFlag) {
              if (trailer !== void 0) {
                throw "extra trailer";
              }
              trailer = trailerParse(data);
              continue;
            }
            if (message2 !== void 0) {
              throw "extra message";
            }
            message2 = parse(data);
          }
          if (trailer === void 0) {
            throw "missing trailer";
          }
          validateTrailer(trailer);
          if (message2 === void 0) {
            throw "missing message";
          }
          return {
            stream: false,
            header: response.headers,
            message: message2,
            trailer
          };
        }
      });
    },
    async stream(service, method, signal, timeoutMs, header, input) {
      var _a2;
      const { serialize, parse } = createClientMethodSerializers(method, useBinaryFormat, options.jsonOptions, options.binaryOptions);
      function parseResponseBody(body, foundStatus, trailerTarget) {
        return __asyncGenerator2(this, arguments, function* parseResponseBody_1() {
          const reader = createEnvelopeReadableStream(body).getReader();
          if (foundStatus) {
            if (!(yield __await2(reader.read())).done) {
              throw "extra data for trailers-only";
            }
            return yield __await2(void 0);
          }
          let trailerReceived = false;
          for (; ; ) {
            const result = yield __await2(reader.read());
            if (result.done) {
              break;
            }
            const { flags, data } = result.value;
            if ((flags & trailerFlag) === trailerFlag) {
              if (trailerReceived) {
                throw "extra trailer";
              }
              trailerReceived = true;
              const trailer = trailerParse(data);
              validateTrailer(trailer);
              trailer.forEach((value, key) => trailerTarget.set(key, value));
              continue;
            }
            if (trailerReceived) {
              throw "extra message";
            }
            yield yield __await2(parse(data));
            continue;
          }
          if (!trailerReceived) {
            throw "missing trailer";
          }
        });
      }
      async function createRequestBody(input2) {
        if (method.kind != MethodKind.ServerStreaming) {
          throw "The fetch API does not support streaming request bodies";
        }
        const r = await input2[Symbol.asyncIterator]().next();
        if (r.done == true) {
          throw "missing request message";
        }
        return encodeEnvelope(0, serialize(r.value));
      }
      return runStreamingCall({
        interceptors: options.interceptors,
        signal,
        timeoutMs,
        req: {
          stream: true,
          service,
          method,
          url: createMethodUrl(options.baseUrl, service, method),
          init: {
            method: "POST",
            credentials: (_a2 = options.credentials) !== null && _a2 !== void 0 ? _a2 : "same-origin",
            redirect: "error",
            mode: "cors"
          },
          header: requestHeader2(useBinaryFormat, timeoutMs, header),
          message: input
        },
        next: async (req) => {
          var _a3;
          const fetch = (_a3 = options.fetch) !== null && _a3 !== void 0 ? _a3 : globalThis.fetch;
          const fRes = await fetch(req.url, Object.assign(Object.assign({}, req.init), { headers: req.header, signal: req.signal, body: await createRequestBody(req.message) }));
          const { foundStatus } = validateResponse2(fRes.status, fRes.headers);
          if (!fRes.body) {
            throw "missing response body";
          }
          const trailer = new Headers();
          const res = Object.assign(Object.assign({}, req), { header: fRes.headers, trailer, message: parseResponseBody(fRes.body, foundStatus, trailer) });
          return res;
        }
      });
    }
  };
}
export {
  createConnectTransport,
  createGrpcWebTransport
};
//# sourceMappingURL=@bufbuild_connect-web.js.map
