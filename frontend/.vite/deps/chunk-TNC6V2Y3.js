// node_modules/@bufbuild/protobuf/dist/esm/private/assert.js
function assert(condition, msg) {
  if (!condition) {
    throw new Error(msg);
  }
}
var FLOAT32_MAX = 34028234663852886e22;
var FLOAT32_MIN = -34028234663852886e22;
var UINT32_MAX = 4294967295;
var INT32_MAX = 2147483647;
var INT32_MIN = -2147483648;
function assertInt32(arg) {
  if (typeof arg !== "number")
    throw new Error("invalid int 32: " + typeof arg);
  if (!Number.isInteger(arg) || arg > INT32_MAX || arg < INT32_MIN)
    throw new Error("invalid int 32: " + arg);
}
function assertUInt32(arg) {
  if (typeof arg !== "number")
    throw new Error("invalid uint 32: " + typeof arg);
  if (!Number.isInteger(arg) || arg > UINT32_MAX || arg < 0)
    throw new Error("invalid uint 32: " + arg);
}
function assertFloat32(arg) {
  if (typeof arg !== "number")
    throw new Error("invalid float 32: " + typeof arg);
  if (!Number.isFinite(arg))
    return;
  if (arg > FLOAT32_MAX || arg < FLOAT32_MIN)
    throw new Error("invalid float 32: " + arg);
}

// node_modules/@bufbuild/protobuf/dist/esm/private/enum.js
var enumTypeSymbol = Symbol("@bufbuild/protobuf/enum-type");
function getEnumType(enumObject) {
  const t = enumObject[enumTypeSymbol];
  assert(t, "missing enum type on enum object");
  return t;
}
function setEnumType(enumObject, typeName, values, opt) {
  enumObject[enumTypeSymbol] = makeEnumType(typeName, values.map((v) => ({
    no: v.no,
    name: v.name,
    localName: enumObject[v.no]
  })), opt);
}
function makeEnumType(typeName, values, _opt) {
  const names = /* @__PURE__ */ Object.create(null);
  const numbers = /* @__PURE__ */ Object.create(null);
  const normalValues = [];
  for (const value of values) {
    const n = normalizeEnumValue(value);
    normalValues.push(n);
    names[value.name] = n;
    numbers[value.no] = n;
  }
  return {
    typeName,
    values: normalValues,
    // We do not surface options at this time
    // options: opt?.options ?? Object.create(null),
    findName(name) {
      return names[name];
    },
    findNumber(no) {
      return numbers[no];
    }
  };
}
function makeEnum(typeName, values, opt) {
  const enumObject = {};
  for (const value of values) {
    const n = normalizeEnumValue(value);
    enumObject[n.localName] = n.no;
    enumObject[n.no] = n.localName;
  }
  setEnumType(enumObject, typeName, values, opt);
  return enumObject;
}
function normalizeEnumValue(value) {
  if ("localName" in value) {
    return value;
  }
  return Object.assign(Object.assign({}, value), { localName: value.name });
}

// node_modules/@bufbuild/protobuf/dist/esm/message.js
var Message = class {
  /**
   * Compare with a message of the same type.
   */
  equals(other) {
    return this.getType().runtime.util.equals(this.getType(), this, other);
  }
  /**
   * Create a deep copy.
   */
  clone() {
    return this.getType().runtime.util.clone(this);
  }
  /**
   * Parse from binary data, merging fields.
   *
   * Repeated fields are appended. Map entries are added, overwriting
   * existing keys.
   *
   * If a message field is already present, it will be merged with the
   * new data.
   */
  fromBinary(bytes, options) {
    const type = this.getType(), format = type.runtime.bin, opt = format.makeReadOptions(options);
    format.readMessage(this, opt.readerFactory(bytes), bytes.byteLength, opt);
    return this;
  }
  /**
   * Parse a message from a JSON value.
   */
  fromJson(jsonValue, options) {
    const type = this.getType(), format = type.runtime.json, opt = format.makeReadOptions(options);
    format.readMessage(type, jsonValue, opt, this);
    return this;
  }
  /**
   * Parse a message from a JSON string.
   */
  fromJsonString(jsonString, options) {
    let json;
    try {
      json = JSON.parse(jsonString);
    } catch (e) {
      throw new Error(`cannot decode ${this.getType().typeName} from JSON: ${e instanceof Error ? e.message : String(e)}`);
    }
    return this.fromJson(json, options);
  }
  /**
   * Serialize the message to binary data.
   */
  toBinary(options) {
    const type = this.getType(), bin = type.runtime.bin, opt = bin.makeWriteOptions(options), writer = opt.writerFactory();
    bin.writeMessage(this, writer, opt);
    return writer.finish();
  }
  /**
   * Serialize the message to a JSON value, a JavaScript value that can be
   * passed to JSON.stringify().
   */
  toJson(options) {
    const type = this.getType(), json = type.runtime.json, opt = json.makeWriteOptions(options);
    return json.writeMessage(this, opt);
  }
  /**
   * Serialize the message to a JSON string.
   */
  toJsonString(options) {
    var _a;
    const value = this.toJson(options);
    return JSON.stringify(value, null, (_a = options === null || options === void 0 ? void 0 : options.prettySpaces) !== null && _a !== void 0 ? _a : 0);
  }
  /**
   * Override for serialization behavior. This will be invoked when calling
   * JSON.stringify on this message (i.e. JSON.stringify(msg)).
   *
   * Note that this will not serialize google.protobuf.Any with a packed
   * message because the protobuf JSON format specifies that it needs to be
   * unpacked, and this is only possible with a type registry to look up the
   * message type.  As a result, attempting to serialize a message with this
   * type will throw an Error.
   *
   * This method is protected because you should not need to invoke it
   * directly -- instead use JSON.stringify or toJsonString for
   * stringified JSON.  Alternatively, if actual JSON is desired, you should
   * use toJson.
   */
  toJSON() {
    return this.toJson({
      emitDefaultValues: true
    });
  }
  /**
   * Retrieve the MessageType of this message - a singleton that represents
   * the protobuf message declaration and provides metadata for reflection-
   * based operations.
   */
  getType() {
    return Object.getPrototypeOf(this).constructor;
  }
};

// node_modules/@bufbuild/protobuf/dist/esm/private/message-type.js
function makeMessageType(runtime, typeName, fields, opt) {
  var _a;
  const localName2 = (_a = opt === null || opt === void 0 ? void 0 : opt.localName) !== null && _a !== void 0 ? _a : typeName.substring(typeName.lastIndexOf(".") + 1);
  const type = {
    [localName2]: function(data) {
      runtime.util.initFields(this);
      runtime.util.initPartial(data, this);
    }
  }[localName2];
  Object.setPrototypeOf(type.prototype, new Message());
  Object.assign(type, {
    runtime,
    typeName,
    fields: runtime.util.newFieldList(fields),
    fromBinary(bytes, options) {
      return new type().fromBinary(bytes, options);
    },
    fromJson(jsonValue, options) {
      return new type().fromJson(jsonValue, options);
    },
    fromJsonString(jsonString, options) {
      return new type().fromJsonString(jsonString, options);
    },
    equals(a, b) {
      return runtime.util.equals(type, a, b);
    }
  });
  return type;
}

// node_modules/@bufbuild/protobuf/dist/esm/private/proto-runtime.js
function makeProtoRuntime(syntax, json, bin, util) {
  return {
    syntax,
    json,
    bin,
    util,
    makeMessageType(typeName, fields, opt) {
      return makeMessageType(this, typeName, fields, opt);
    },
    makeEnum,
    makeEnumType,
    getEnumType
  };
}

// node_modules/@bufbuild/protobuf/dist/esm/field.js
var ScalarType;
(function(ScalarType2) {
  ScalarType2[ScalarType2["DOUBLE"] = 1] = "DOUBLE";
  ScalarType2[ScalarType2["FLOAT"] = 2] = "FLOAT";
  ScalarType2[ScalarType2["INT64"] = 3] = "INT64";
  ScalarType2[ScalarType2["UINT64"] = 4] = "UINT64";
  ScalarType2[ScalarType2["INT32"] = 5] = "INT32";
  ScalarType2[ScalarType2["FIXED64"] = 6] = "FIXED64";
  ScalarType2[ScalarType2["FIXED32"] = 7] = "FIXED32";
  ScalarType2[ScalarType2["BOOL"] = 8] = "BOOL";
  ScalarType2[ScalarType2["STRING"] = 9] = "STRING";
  ScalarType2[ScalarType2["BYTES"] = 12] = "BYTES";
  ScalarType2[ScalarType2["UINT32"] = 13] = "UINT32";
  ScalarType2[ScalarType2["SFIXED32"] = 15] = "SFIXED32";
  ScalarType2[ScalarType2["SFIXED64"] = 16] = "SFIXED64";
  ScalarType2[ScalarType2["SINT32"] = 17] = "SINT32";
  ScalarType2[ScalarType2["SINT64"] = 18] = "SINT64";
})(ScalarType || (ScalarType = {}));

// node_modules/@bufbuild/protobuf/dist/esm/google/varint.js
function varint64read() {
  let lowBits = 0;
  let highBits = 0;
  for (let shift = 0; shift < 28; shift += 7) {
    let b = this.buf[this.pos++];
    lowBits |= (b & 127) << shift;
    if ((b & 128) == 0) {
      this.assertBounds();
      return [lowBits, highBits];
    }
  }
  let middleByte = this.buf[this.pos++];
  lowBits |= (middleByte & 15) << 28;
  highBits = (middleByte & 112) >> 4;
  if ((middleByte & 128) == 0) {
    this.assertBounds();
    return [lowBits, highBits];
  }
  for (let shift = 3; shift <= 31; shift += 7) {
    let b = this.buf[this.pos++];
    highBits |= (b & 127) << shift;
    if ((b & 128) == 0) {
      this.assertBounds();
      return [lowBits, highBits];
    }
  }
  throw new Error("invalid varint");
}
function varint64write(lo, hi, bytes) {
  for (let i = 0; i < 28; i = i + 7) {
    const shift = lo >>> i;
    const hasNext = !(shift >>> 7 == 0 && hi == 0);
    const byte = (hasNext ? shift | 128 : shift) & 255;
    bytes.push(byte);
    if (!hasNext) {
      return;
    }
  }
  const splitBits = lo >>> 28 & 15 | (hi & 7) << 4;
  const hasMoreBits = !(hi >> 3 == 0);
  bytes.push((hasMoreBits ? splitBits | 128 : splitBits) & 255);
  if (!hasMoreBits) {
    return;
  }
  for (let i = 3; i < 31; i = i + 7) {
    const shift = hi >>> i;
    const hasNext = !(shift >>> 7 == 0);
    const byte = (hasNext ? shift | 128 : shift) & 255;
    bytes.push(byte);
    if (!hasNext) {
      return;
    }
  }
  bytes.push(hi >>> 31 & 1);
}
var TWO_PWR_32_DBL = 4294967296;
function int64FromString(dec) {
  const minus = dec[0] === "-";
  if (minus) {
    dec = dec.slice(1);
  }
  const base = 1e6;
  let lowBits = 0;
  let highBits = 0;
  function add1e6digit(begin, end) {
    const digit1e6 = Number(dec.slice(begin, end));
    highBits *= base;
    lowBits = lowBits * base + digit1e6;
    if (lowBits >= TWO_PWR_32_DBL) {
      highBits = highBits + (lowBits / TWO_PWR_32_DBL | 0);
      lowBits = lowBits % TWO_PWR_32_DBL;
    }
  }
  add1e6digit(-24, -18);
  add1e6digit(-18, -12);
  add1e6digit(-12, -6);
  add1e6digit(-6);
  return minus ? negate(lowBits, highBits) : newBits(lowBits, highBits);
}
function int64ToString(lo, hi) {
  let bits = newBits(lo, hi);
  const negative = bits.hi & 2147483648;
  if (negative) {
    bits = negate(bits.lo, bits.hi);
  }
  const result = uInt64ToString(bits.lo, bits.hi);
  return negative ? "-" + result : result;
}
function uInt64ToString(lo, hi) {
  ({ lo, hi } = toUnsigned(lo, hi));
  if (hi <= 2097151) {
    return String(TWO_PWR_32_DBL * hi + lo);
  }
  const low = lo & 16777215;
  const mid = (lo >>> 24 | hi << 8) & 16777215;
  const high = hi >> 16 & 65535;
  let digitA = low + mid * 6777216 + high * 6710656;
  let digitB = mid + high * 8147497;
  let digitC = high * 2;
  const base = 1e7;
  if (digitA >= base) {
    digitB += Math.floor(digitA / base);
    digitA %= base;
  }
  if (digitB >= base) {
    digitC += Math.floor(digitB / base);
    digitB %= base;
  }
  return digitC.toString() + decimalFrom1e7WithLeadingZeros(digitB) + decimalFrom1e7WithLeadingZeros(digitA);
}
function toUnsigned(lo, hi) {
  return { lo: lo >>> 0, hi: hi >>> 0 };
}
function newBits(lo, hi) {
  return { lo: lo | 0, hi: hi | 0 };
}
function negate(lowBits, highBits) {
  highBits = ~highBits;
  if (lowBits) {
    lowBits = ~lowBits + 1;
  } else {
    highBits += 1;
  }
  return newBits(lowBits, highBits);
}
var decimalFrom1e7WithLeadingZeros = (digit1e7) => {
  const partial = String(digit1e7);
  return "0000000".slice(partial.length) + partial;
};
function varint32write(value, bytes) {
  if (value >= 0) {
    while (value > 127) {
      bytes.push(value & 127 | 128);
      value = value >>> 7;
    }
    bytes.push(value);
  } else {
    for (let i = 0; i < 9; i++) {
      bytes.push(value & 127 | 128);
      value = value >> 7;
    }
    bytes.push(1);
  }
}
function varint32read() {
  let b = this.buf[this.pos++];
  let result = b & 127;
  if ((b & 128) == 0) {
    this.assertBounds();
    return result;
  }
  b = this.buf[this.pos++];
  result |= (b & 127) << 7;
  if ((b & 128) == 0) {
    this.assertBounds();
    return result;
  }
  b = this.buf[this.pos++];
  result |= (b & 127) << 14;
  if ((b & 128) == 0) {
    this.assertBounds();
    return result;
  }
  b = this.buf[this.pos++];
  result |= (b & 127) << 21;
  if ((b & 128) == 0) {
    this.assertBounds();
    return result;
  }
  b = this.buf[this.pos++];
  result |= (b & 15) << 28;
  for (let readBytes = 5; (b & 128) !== 0 && readBytes < 10; readBytes++)
    b = this.buf[this.pos++];
  if ((b & 128) != 0)
    throw new Error("invalid varint");
  this.assertBounds();
  return result >>> 0;
}

// node_modules/@bufbuild/protobuf/dist/esm/proto-int64.js
function makeInt64Support() {
  const dv = new DataView(new ArrayBuffer(8));
  const ok = typeof BigInt === "function" && typeof dv.getBigInt64 === "function" && typeof dv.getBigUint64 === "function" && typeof dv.setBigInt64 === "function" && typeof dv.setBigUint64 === "function" && (typeof process != "object" || typeof process.env != "object" || process.env.BUF_BIGINT_DISABLE !== "1");
  if (ok) {
    const MIN = BigInt("-9223372036854775808"), MAX = BigInt("9223372036854775807"), UMIN = BigInt("0"), UMAX = BigInt("18446744073709551615");
    return {
      zero: BigInt(0),
      supported: true,
      parse(value) {
        const bi = typeof value == "bigint" ? value : BigInt(value);
        if (bi > MAX || bi < MIN) {
          throw new Error(`int64 invalid: ${value}`);
        }
        return bi;
      },
      uParse(value) {
        const bi = typeof value == "bigint" ? value : BigInt(value);
        if (bi > UMAX || bi < UMIN) {
          throw new Error(`uint64 invalid: ${value}`);
        }
        return bi;
      },
      enc(value) {
        dv.setBigInt64(0, this.parse(value), true);
        return {
          lo: dv.getInt32(0, true),
          hi: dv.getInt32(4, true)
        };
      },
      uEnc(value) {
        dv.setBigInt64(0, this.uParse(value), true);
        return {
          lo: dv.getInt32(0, true),
          hi: dv.getInt32(4, true)
        };
      },
      dec(lo, hi) {
        dv.setInt32(0, lo, true);
        dv.setInt32(4, hi, true);
        return dv.getBigInt64(0, true);
      },
      uDec(lo, hi) {
        dv.setInt32(0, lo, true);
        dv.setInt32(4, hi, true);
        return dv.getBigUint64(0, true);
      }
    };
  }
  const assertInt64String = (value) => assert(/^-?[0-9]+$/.test(value), `int64 invalid: ${value}`);
  const assertUInt64String = (value) => assert(/^[0-9]+$/.test(value), `uint64 invalid: ${value}`);
  return {
    zero: "0",
    supported: false,
    parse(value) {
      if (typeof value != "string") {
        value = value.toString();
      }
      assertInt64String(value);
      return value;
    },
    uParse(value) {
      if (typeof value != "string") {
        value = value.toString();
      }
      assertUInt64String(value);
      return value;
    },
    enc(value) {
      if (typeof value != "string") {
        value = value.toString();
      }
      assertInt64String(value);
      return int64FromString(value);
    },
    uEnc(value) {
      if (typeof value != "string") {
        value = value.toString();
      }
      assertUInt64String(value);
      return int64FromString(value);
    },
    dec(lo, hi) {
      return int64ToString(lo, hi);
    },
    uDec(lo, hi) {
      return uInt64ToString(lo, hi);
    }
  };
}
var protoInt64 = makeInt64Support();

// node_modules/@bufbuild/protobuf/dist/esm/binary-encoding.js
var WireType;
(function(WireType2) {
  WireType2[WireType2["Varint"] = 0] = "Varint";
  WireType2[WireType2["Bit64"] = 1] = "Bit64";
  WireType2[WireType2["LengthDelimited"] = 2] = "LengthDelimited";
  WireType2[WireType2["StartGroup"] = 3] = "StartGroup";
  WireType2[WireType2["EndGroup"] = 4] = "EndGroup";
  WireType2[WireType2["Bit32"] = 5] = "Bit32";
})(WireType || (WireType = {}));
var BinaryWriter = class {
  constructor(textEncoder) {
    this.stack = [];
    this.textEncoder = textEncoder !== null && textEncoder !== void 0 ? textEncoder : new TextEncoder();
    this.chunks = [];
    this.buf = [];
  }
  /**
   * Return all bytes written and reset this writer.
   */
  finish() {
    this.chunks.push(new Uint8Array(this.buf));
    let len = 0;
    for (let i = 0; i < this.chunks.length; i++)
      len += this.chunks[i].length;
    let bytes = new Uint8Array(len);
    let offset = 0;
    for (let i = 0; i < this.chunks.length; i++) {
      bytes.set(this.chunks[i], offset);
      offset += this.chunks[i].length;
    }
    this.chunks = [];
    return bytes;
  }
  /**
   * Start a new fork for length-delimited data like a message
   * or a packed repeated field.
   *
   * Must be joined later with `join()`.
   */
  fork() {
    this.stack.push({ chunks: this.chunks, buf: this.buf });
    this.chunks = [];
    this.buf = [];
    return this;
  }
  /**
   * Join the last fork. Write its length and bytes, then
   * return to the previous state.
   */
  join() {
    let chunk = this.finish();
    let prev = this.stack.pop();
    if (!prev)
      throw new Error("invalid state, fork stack empty");
    this.chunks = prev.chunks;
    this.buf = prev.buf;
    this.uint32(chunk.byteLength);
    return this.raw(chunk);
  }
  /**
   * Writes a tag (field number and wire type).
   *
   * Equivalent to `uint32( (fieldNo << 3 | type) >>> 0 )`.
   *
   * Generated code should compute the tag ahead of time and call `uint32()`.
   */
  tag(fieldNo, type) {
    return this.uint32((fieldNo << 3 | type) >>> 0);
  }
  /**
   * Write a chunk of raw bytes.
   */
  raw(chunk) {
    if (this.buf.length) {
      this.chunks.push(new Uint8Array(this.buf));
      this.buf = [];
    }
    this.chunks.push(chunk);
    return this;
  }
  /**
   * Write a `uint32` value, an unsigned 32 bit varint.
   */
  uint32(value) {
    assertUInt32(value);
    while (value > 127) {
      this.buf.push(value & 127 | 128);
      value = value >>> 7;
    }
    this.buf.push(value);
    return this;
  }
  /**
   * Write a `int32` value, a signed 32 bit varint.
   */
  int32(value) {
    assertInt32(value);
    varint32write(value, this.buf);
    return this;
  }
  /**
   * Write a `bool` value, a variant.
   */
  bool(value) {
    this.buf.push(value ? 1 : 0);
    return this;
  }
  /**
   * Write a `bytes` value, length-delimited arbitrary data.
   */
  bytes(value) {
    this.uint32(value.byteLength);
    return this.raw(value);
  }
  /**
   * Write a `string` value, length-delimited data converted to UTF-8 text.
   */
  string(value) {
    let chunk = this.textEncoder.encode(value);
    this.uint32(chunk.byteLength);
    return this.raw(chunk);
  }
  /**
   * Write a `float` value, 32-bit floating point number.
   */
  float(value) {
    assertFloat32(value);
    let chunk = new Uint8Array(4);
    new DataView(chunk.buffer).setFloat32(0, value, true);
    return this.raw(chunk);
  }
  /**
   * Write a `double` value, a 64-bit floating point number.
   */
  double(value) {
    let chunk = new Uint8Array(8);
    new DataView(chunk.buffer).setFloat64(0, value, true);
    return this.raw(chunk);
  }
  /**
   * Write a `fixed32` value, an unsigned, fixed-length 32-bit integer.
   */
  fixed32(value) {
    assertUInt32(value);
    let chunk = new Uint8Array(4);
    new DataView(chunk.buffer).setUint32(0, value, true);
    return this.raw(chunk);
  }
  /**
   * Write a `sfixed32` value, a signed, fixed-length 32-bit integer.
   */
  sfixed32(value) {
    assertInt32(value);
    let chunk = new Uint8Array(4);
    new DataView(chunk.buffer).setInt32(0, value, true);
    return this.raw(chunk);
  }
  /**
   * Write a `sint32` value, a signed, zigzag-encoded 32-bit varint.
   */
  sint32(value) {
    assertInt32(value);
    value = (value << 1 ^ value >> 31) >>> 0;
    varint32write(value, this.buf);
    return this;
  }
  /**
   * Write a `fixed64` value, a signed, fixed-length 64-bit integer.
   */
  sfixed64(value) {
    let chunk = new Uint8Array(8), view = new DataView(chunk.buffer), tc = protoInt64.enc(value);
    view.setInt32(0, tc.lo, true);
    view.setInt32(4, tc.hi, true);
    return this.raw(chunk);
  }
  /**
   * Write a `fixed64` value, an unsigned, fixed-length 64 bit integer.
   */
  fixed64(value) {
    let chunk = new Uint8Array(8), view = new DataView(chunk.buffer), tc = protoInt64.uEnc(value);
    view.setInt32(0, tc.lo, true);
    view.setInt32(4, tc.hi, true);
    return this.raw(chunk);
  }
  /**
   * Write a `int64` value, a signed 64-bit varint.
   */
  int64(value) {
    let tc = protoInt64.enc(value);
    varint64write(tc.lo, tc.hi, this.buf);
    return this;
  }
  /**
   * Write a `sint64` value, a signed, zig-zag-encoded 64-bit varint.
   */
  sint64(value) {
    let tc = protoInt64.enc(value), sign = tc.hi >> 31, lo = tc.lo << 1 ^ sign, hi = (tc.hi << 1 | tc.lo >>> 31) ^ sign;
    varint64write(lo, hi, this.buf);
    return this;
  }
  /**
   * Write a `uint64` value, an unsigned 64-bit varint.
   */
  uint64(value) {
    let tc = protoInt64.uEnc(value);
    varint64write(tc.lo, tc.hi, this.buf);
    return this;
  }
};
var BinaryReader = class {
  constructor(buf, textDecoder) {
    this.varint64 = varint64read;
    this.uint32 = varint32read;
    this.buf = buf;
    this.len = buf.length;
    this.pos = 0;
    this.view = new DataView(buf.buffer, buf.byteOffset, buf.byteLength);
    this.textDecoder = textDecoder !== null && textDecoder !== void 0 ? textDecoder : new TextDecoder();
  }
  /**
   * Reads a tag - field number and wire type.
   */
  tag() {
    let tag = this.uint32(), fieldNo = tag >>> 3, wireType = tag & 7;
    if (fieldNo <= 0 || wireType < 0 || wireType > 5)
      throw new Error("illegal tag: field no " + fieldNo + " wire type " + wireType);
    return [fieldNo, wireType];
  }
  /**
   * Skip one element on the wire and return the skipped data.
   * Supports WireType.StartGroup since v2.0.0-alpha.23.
   */
  skip(wireType) {
    let start = this.pos;
    switch (wireType) {
      case WireType.Varint:
        while (this.buf[this.pos++] & 128) {
        }
        break;
      case WireType.Bit64:
        this.pos += 4;
      case WireType.Bit32:
        this.pos += 4;
        break;
      case WireType.LengthDelimited:
        let len = this.uint32();
        this.pos += len;
        break;
      case WireType.StartGroup:
        let t;
        while ((t = this.tag()[1]) !== WireType.EndGroup) {
          this.skip(t);
        }
        break;
      default:
        throw new Error("cant skip wire type " + wireType);
    }
    this.assertBounds();
    return this.buf.subarray(start, this.pos);
  }
  /**
   * Throws error if position in byte array is out of range.
   */
  assertBounds() {
    if (this.pos > this.len)
      throw new RangeError("premature EOF");
  }
  /**
   * Read a `int32` field, a signed 32 bit varint.
   */
  int32() {
    return this.uint32() | 0;
  }
  /**
   * Read a `sint32` field, a signed, zigzag-encoded 32-bit varint.
   */
  sint32() {
    let zze = this.uint32();
    return zze >>> 1 ^ -(zze & 1);
  }
  /**
   * Read a `int64` field, a signed 64-bit varint.
   */
  int64() {
    return protoInt64.dec(...this.varint64());
  }
  /**
   * Read a `uint64` field, an unsigned 64-bit varint.
   */
  uint64() {
    return protoInt64.uDec(...this.varint64());
  }
  /**
   * Read a `sint64` field, a signed, zig-zag-encoded 64-bit varint.
   */
  sint64() {
    let [lo, hi] = this.varint64();
    let s = -(lo & 1);
    lo = (lo >>> 1 | (hi & 1) << 31) ^ s;
    hi = hi >>> 1 ^ s;
    return protoInt64.dec(lo, hi);
  }
  /**
   * Read a `bool` field, a variant.
   */
  bool() {
    let [lo, hi] = this.varint64();
    return lo !== 0 || hi !== 0;
  }
  /**
   * Read a `fixed32` field, an unsigned, fixed-length 32-bit integer.
   */
  fixed32() {
    return this.view.getUint32((this.pos += 4) - 4, true);
  }
  /**
   * Read a `sfixed32` field, a signed, fixed-length 32-bit integer.
   */
  sfixed32() {
    return this.view.getInt32((this.pos += 4) - 4, true);
  }
  /**
   * Read a `fixed64` field, an unsigned, fixed-length 64 bit integer.
   */
  fixed64() {
    return protoInt64.uDec(this.sfixed32(), this.sfixed32());
  }
  /**
   * Read a `fixed64` field, a signed, fixed-length 64-bit integer.
   */
  sfixed64() {
    return protoInt64.dec(this.sfixed32(), this.sfixed32());
  }
  /**
   * Read a `float` field, 32-bit floating point number.
   */
  float() {
    return this.view.getFloat32((this.pos += 4) - 4, true);
  }
  /**
   * Read a `double` field, a 64-bit floating point number.
   */
  double() {
    return this.view.getFloat64((this.pos += 8) - 8, true);
  }
  /**
   * Read a `bytes` field, length-delimited arbitrary data.
   */
  bytes() {
    let len = this.uint32(), start = this.pos;
    this.pos += len;
    this.assertBounds();
    return this.buf.subarray(start, start + len);
  }
  /**
   * Read a `string` field, length-delimited data converted to UTF-8 text.
   */
  string() {
    return this.textDecoder.decode(this.bytes());
  }
};

// node_modules/@bufbuild/protobuf/dist/esm/private/field-wrapper.js
function wrapField(type, value) {
  if (value instanceof Message || !type.fieldWrapper) {
    return value;
  }
  return type.fieldWrapper.wrapField(value);
}
function getUnwrappedFieldType(field) {
  if (field.fieldKind !== "message") {
    return void 0;
  }
  if (field.repeated) {
    return void 0;
  }
  if (field.oneof != void 0) {
    return void 0;
  }
  return wktWrapperToScalarType[field.message.typeName];
}
var wktWrapperToScalarType = {
  "google.protobuf.DoubleValue": ScalarType.DOUBLE,
  "google.protobuf.FloatValue": ScalarType.FLOAT,
  "google.protobuf.Int64Value": ScalarType.INT64,
  "google.protobuf.UInt64Value": ScalarType.UINT64,
  "google.protobuf.Int32Value": ScalarType.INT32,
  "google.protobuf.UInt32Value": ScalarType.UINT32,
  "google.protobuf.BoolValue": ScalarType.BOOL,
  "google.protobuf.StringValue": ScalarType.STRING,
  "google.protobuf.BytesValue": ScalarType.BYTES
};

// node_modules/@bufbuild/protobuf/dist/esm/private/scalars.js
function scalarEquals(type, a, b) {
  if (a === b) {
    return true;
  }
  if (type == ScalarType.BYTES) {
    if (!(a instanceof Uint8Array) || !(b instanceof Uint8Array)) {
      return false;
    }
    if (a.length !== b.length) {
      return false;
    }
    for (let i = 0; i < a.length; i++) {
      if (a[i] !== b[i]) {
        return false;
      }
    }
    return true;
  }
  switch (type) {
    case ScalarType.UINT64:
    case ScalarType.FIXED64:
    case ScalarType.INT64:
    case ScalarType.SFIXED64:
    case ScalarType.SINT64:
      return a == b;
  }
  return false;
}
function scalarDefaultValue(type) {
  switch (type) {
    case ScalarType.BOOL:
      return false;
    case ScalarType.UINT64:
    case ScalarType.FIXED64:
    case ScalarType.INT64:
    case ScalarType.SFIXED64:
    case ScalarType.SINT64:
      return protoInt64.zero;
    case ScalarType.DOUBLE:
    case ScalarType.FLOAT:
      return 0;
    case ScalarType.BYTES:
      return new Uint8Array(0);
    case ScalarType.STRING:
      return "";
    default:
      return 0;
  }
}
function scalarTypeInfo(type, value) {
  const isUndefined = value === void 0;
  let wireType = WireType.Varint;
  let isIntrinsicDefault = value === 0;
  switch (type) {
    case ScalarType.STRING:
      isIntrinsicDefault = isUndefined || !value.length;
      wireType = WireType.LengthDelimited;
      break;
    case ScalarType.BOOL:
      isIntrinsicDefault = value === false;
      break;
    case ScalarType.DOUBLE:
      wireType = WireType.Bit64;
      break;
    case ScalarType.FLOAT:
      wireType = WireType.Bit32;
      break;
    case ScalarType.INT64:
      isIntrinsicDefault = isUndefined || value == 0;
      break;
    case ScalarType.UINT64:
      isIntrinsicDefault = isUndefined || value == 0;
      break;
    case ScalarType.FIXED64:
      isIntrinsicDefault = isUndefined || value == 0;
      wireType = WireType.Bit64;
      break;
    case ScalarType.BYTES:
      isIntrinsicDefault = isUndefined || !value.byteLength;
      wireType = WireType.LengthDelimited;
      break;
    case ScalarType.FIXED32:
      wireType = WireType.Bit32;
      break;
    case ScalarType.SFIXED32:
      wireType = WireType.Bit32;
      break;
    case ScalarType.SFIXED64:
      isIntrinsicDefault = isUndefined || value == 0;
      wireType = WireType.Bit64;
      break;
    case ScalarType.SINT64:
      isIntrinsicDefault = isUndefined || value == 0;
      break;
  }
  const method = ScalarType[type].toLowerCase();
  return [wireType, method, isUndefined || isIntrinsicDefault];
}

// node_modules/@bufbuild/protobuf/dist/esm/private/binary-format-common.js
var unknownFieldsSymbol = Symbol("@bufbuild/protobuf/unknown-fields");
var readDefaults = {
  readUnknownFields: true,
  readerFactory: (bytes) => new BinaryReader(bytes)
};
var writeDefaults = {
  writeUnknownFields: true,
  writerFactory: () => new BinaryWriter()
};
function makeReadOptions(options) {
  return options ? Object.assign(Object.assign({}, readDefaults), options) : readDefaults;
}
function makeWriteOptions(options) {
  return options ? Object.assign(Object.assign({}, writeDefaults), options) : writeDefaults;
}
function makeBinaryFormatCommon() {
  return {
    makeReadOptions,
    makeWriteOptions,
    listUnknownFields(message) {
      var _a;
      return (_a = message[unknownFieldsSymbol]) !== null && _a !== void 0 ? _a : [];
    },
    discardUnknownFields(message) {
      delete message[unknownFieldsSymbol];
    },
    writeUnknownFields(message, writer) {
      const m = message;
      const c = m[unknownFieldsSymbol];
      if (c) {
        for (const f of c) {
          writer.tag(f.no, f.wireType).raw(f.data);
        }
      }
    },
    onUnknownField(message, no, wireType, data) {
      const m = message;
      if (!Array.isArray(m[unknownFieldsSymbol])) {
        m[unknownFieldsSymbol] = [];
      }
      m[unknownFieldsSymbol].push({ no, wireType, data });
    },
    readMessage(message, reader, length, options) {
      const type = message.getType();
      const end = length === void 0 ? reader.len : reader.pos + length;
      while (reader.pos < end) {
        const [fieldNo, wireType] = reader.tag(), field = type.fields.find(fieldNo);
        if (!field) {
          const data = reader.skip(wireType);
          if (options.readUnknownFields) {
            this.onUnknownField(message, fieldNo, wireType, data);
          }
          continue;
        }
        let target = message, repeated = field.repeated, localName2 = field.localName;
        if (field.oneof) {
          target = target[field.oneof.localName];
          if (target.case != localName2) {
            delete target.value;
          }
          target.case = localName2;
          localName2 = "value";
        }
        switch (field.kind) {
          case "scalar":
          case "enum":
            const scalarType = field.kind == "enum" ? ScalarType.INT32 : field.T;
            if (repeated) {
              let arr = target[localName2];
              if (wireType == WireType.LengthDelimited && scalarType != ScalarType.STRING && scalarType != ScalarType.BYTES) {
                let e = reader.uint32() + reader.pos;
                while (reader.pos < e) {
                  arr.push(readScalar(reader, scalarType));
                }
              } else {
                arr.push(readScalar(reader, scalarType));
              }
            } else {
              target[localName2] = readScalar(reader, scalarType);
            }
            break;
          case "message":
            const messageType = field.T;
            if (repeated) {
              target[localName2].push(readMessageField(reader, new messageType(), options));
            } else {
              if (target[localName2] instanceof Message) {
                readMessageField(reader, target[localName2], options);
              } else {
                target[localName2] = readMessageField(reader, new messageType(), options);
                if (messageType.fieldWrapper && !field.oneof && !field.repeated) {
                  target[localName2] = messageType.fieldWrapper.unwrapField(target[localName2]);
                }
              }
            }
            break;
          case "map":
            let [mapKey, mapVal] = readMapEntry(field, reader, options);
            target[localName2][mapKey] = mapVal;
            break;
        }
      }
    }
  };
}
function readMessageField(reader, message, options) {
  const format = message.getType().runtime.bin;
  format.readMessage(message, reader, reader.uint32(), options);
  return message;
}
function readMapEntry(field, reader, options) {
  const length = reader.uint32(), end = reader.pos + length;
  let key, val;
  while (reader.pos < end) {
    let [fieldNo] = reader.tag();
    switch (fieldNo) {
      case 1:
        key = readScalar(reader, field.K);
        break;
      case 2:
        switch (field.V.kind) {
          case "scalar":
            val = readScalar(reader, field.V.T);
            break;
          case "enum":
            val = reader.int32();
            break;
          case "message":
            val = readMessageField(reader, new field.V.T(), options);
            break;
        }
        break;
    }
  }
  if (key === void 0) {
    let keyRaw = scalarDefaultValue(field.K);
    key = field.K == ScalarType.BOOL ? keyRaw.toString() : keyRaw;
  }
  if (typeof key != "string" && typeof key != "number") {
    key = key.toString();
  }
  if (val === void 0) {
    switch (field.V.kind) {
      case "scalar":
        val = scalarDefaultValue(field.V.T);
        break;
      case "enum":
        val = 0;
        break;
      case "message":
        val = new field.V.T();
        break;
    }
  }
  return [key, val];
}
function readScalar(reader, type) {
  switch (type) {
    case ScalarType.STRING:
      return reader.string();
    case ScalarType.BOOL:
      return reader.bool();
    case ScalarType.DOUBLE:
      return reader.double();
    case ScalarType.FLOAT:
      return reader.float();
    case ScalarType.INT32:
      return reader.int32();
    case ScalarType.INT64:
      return reader.int64();
    case ScalarType.UINT64:
      return reader.uint64();
    case ScalarType.FIXED64:
      return reader.fixed64();
    case ScalarType.BYTES:
      return reader.bytes();
    case ScalarType.FIXED32:
      return reader.fixed32();
    case ScalarType.SFIXED32:
      return reader.sfixed32();
    case ScalarType.SFIXED64:
      return reader.sfixed64();
    case ScalarType.SINT64:
      return reader.sint64();
    case ScalarType.UINT32:
      return reader.uint32();
    case ScalarType.SINT32:
      return reader.sint32();
  }
}
function writeMapEntry(writer, options, field, key, value) {
  writer.tag(field.no, WireType.LengthDelimited);
  writer.fork();
  let keyValue = key;
  switch (field.K) {
    case ScalarType.INT32:
    case ScalarType.FIXED32:
    case ScalarType.UINT32:
    case ScalarType.SFIXED32:
    case ScalarType.SINT32:
      keyValue = Number.parseInt(key);
      break;
    case ScalarType.BOOL:
      assert(key == "true" || key == "false");
      keyValue = key == "true";
      break;
  }
  writeScalar(writer, field.K, 1, keyValue, true);
  switch (field.V.kind) {
    case "scalar":
      writeScalar(writer, field.V.T, 2, value, true);
      break;
    case "enum":
      writeScalar(writer, ScalarType.INT32, 2, value, true);
      break;
    case "message":
      writeMessageField(writer, options, field.V.T, 2, value);
      break;
  }
  writer.join();
}
function writeMessageField(writer, options, type, fieldNo, value) {
  if (value !== void 0) {
    const message = wrapField(type, value);
    writer.tag(fieldNo, WireType.LengthDelimited).bytes(message.toBinary(options));
  }
}
function writeScalar(writer, type, fieldNo, value, emitIntrinsicDefault) {
  let [wireType, method, isIntrinsicDefault] = scalarTypeInfo(type, value);
  if (!isIntrinsicDefault || emitIntrinsicDefault) {
    writer.tag(fieldNo, wireType)[method](value);
  }
}
function writePacked(writer, type, fieldNo, value) {
  if (!value.length) {
    return;
  }
  writer.tag(fieldNo, WireType.LengthDelimited).fork();
  let [, method] = scalarTypeInfo(type);
  for (let i = 0; i < value.length; i++) {
    writer[method](value[i]);
  }
  writer.join();
}

// node_modules/@bufbuild/protobuf/dist/esm/private/binary-format-proto3.js
function makeBinaryFormatProto3() {
  return Object.assign(Object.assign({}, makeBinaryFormatCommon()), { writeMessage(message, writer, options) {
    const type = message.getType();
    for (const field of type.fields.byNumber()) {
      let value, repeated = field.repeated, localName2 = field.localName;
      if (field.oneof) {
        const oneof = message[field.oneof.localName];
        if (oneof.case !== localName2) {
          continue;
        }
        value = oneof.value;
      } else {
        value = message[localName2];
      }
      switch (field.kind) {
        case "scalar":
        case "enum":
          let scalarType = field.kind == "enum" ? ScalarType.INT32 : field.T;
          if (repeated) {
            if (field.packed) {
              writePacked(writer, scalarType, field.no, value);
            } else {
              for (const item of value) {
                writeScalar(writer, scalarType, field.no, item, true);
              }
            }
          } else {
            if (value !== void 0) {
              writeScalar(writer, scalarType, field.no, value, !!field.oneof || field.opt);
            }
          }
          break;
        case "message":
          if (repeated) {
            for (const item of value) {
              writeMessageField(writer, options, field.T, field.no, item);
            }
          } else {
            writeMessageField(writer, options, field.T, field.no, value);
          }
          break;
        case "map":
          for (const [key, val] of Object.entries(value)) {
            writeMapEntry(writer, options, field, key, val);
          }
          break;
      }
    }
    if (options.writeUnknownFields) {
      this.writeUnknownFields(message, writer);
    }
    return writer;
  } });
}

// node_modules/@bufbuild/protobuf/dist/esm/proto-base64.js
var encTable = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/".split("");
var decTable = [];
for (let i = 0; i < encTable.length; i++)
  decTable[encTable[i].charCodeAt(0)] = i;
decTable["-".charCodeAt(0)] = encTable.indexOf("+");
decTable["_".charCodeAt(0)] = encTable.indexOf("/");
var protoBase64 = {
  /**
   * Decodes a base64 string to a byte array.
   *
   * - ignores white-space, including line breaks and tabs
   * - allows inner padding (can decode concatenated base64 strings)
   * - does not require padding
   * - understands base64url encoding:
   *   "-" instead of "+",
   *   "_" instead of "/",
   *   no padding
   */
  dec(base64Str) {
    let es = base64Str.length * 3 / 4;
    if (base64Str[base64Str.length - 2] == "=")
      es -= 2;
    else if (base64Str[base64Str.length - 1] == "=")
      es -= 1;
    let bytes = new Uint8Array(es), bytePos = 0, groupPos = 0, b, p = 0;
    for (let i = 0; i < base64Str.length; i++) {
      b = decTable[base64Str.charCodeAt(i)];
      if (b === void 0) {
        switch (base64Str[i]) {
          case "=":
            groupPos = 0;
          case "\n":
          case "\r":
          case "	":
          case " ":
            continue;
          default:
            throw Error("invalid base64 string.");
        }
      }
      switch (groupPos) {
        case 0:
          p = b;
          groupPos = 1;
          break;
        case 1:
          bytes[bytePos++] = p << 2 | (b & 48) >> 4;
          p = b;
          groupPos = 2;
          break;
        case 2:
          bytes[bytePos++] = (p & 15) << 4 | (b & 60) >> 2;
          p = b;
          groupPos = 3;
          break;
        case 3:
          bytes[bytePos++] = (p & 3) << 6 | b;
          groupPos = 0;
          break;
      }
    }
    if (groupPos == 1)
      throw Error("invalid base64 string.");
    return bytes.subarray(0, bytePos);
  },
  /**
   * Encode a byte array to a base64 string.
   */
  enc(bytes) {
    let base64 = "", groupPos = 0, b, p = 0;
    for (let i = 0; i < bytes.length; i++) {
      b = bytes[i];
      switch (groupPos) {
        case 0:
          base64 += encTable[b >> 2];
          p = (b & 3) << 4;
          groupPos = 1;
          break;
        case 1:
          base64 += encTable[p | b >> 4];
          p = (b & 15) << 2;
          groupPos = 2;
          break;
        case 2:
          base64 += encTable[p | b >> 6];
          base64 += encTable[b & 63];
          groupPos = 0;
          break;
      }
    }
    if (groupPos) {
      base64 += encTable[p];
      base64 += "=";
      if (groupPos == 1)
        base64 += "=";
    }
    return base64;
  }
};

// node_modules/@bufbuild/protobuf/dist/esm/private/json-format-common.js
var jsonReadDefaults = {
  ignoreUnknownFields: false
};
var jsonWriteDefaults = {
  emitDefaultValues: false,
  enumAsInteger: false,
  useProtoFieldName: false,
  prettySpaces: 0
};
function makeReadOptions2(options) {
  return options ? Object.assign(Object.assign({}, jsonReadDefaults), options) : jsonReadDefaults;
}
function makeWriteOptions2(options) {
  return options ? Object.assign(Object.assign({}, jsonWriteDefaults), options) : jsonWriteDefaults;
}
function makeJsonFormatCommon(makeWriteField) {
  const writeField = makeWriteField(writeEnum, writeScalar2);
  return {
    makeReadOptions: makeReadOptions2,
    makeWriteOptions: makeWriteOptions2,
    readMessage(type, json, options, message) {
      if (json == null || Array.isArray(json) || typeof json != "object") {
        throw new Error(`cannot decode message ${type.typeName} from JSON: ${this.debug(json)}`);
      }
      message = message !== null && message !== void 0 ? message : new type();
      const oneofSeen = {};
      for (const [jsonKey, jsonValue] of Object.entries(json)) {
        const field = type.fields.findJsonName(jsonKey);
        if (!field) {
          if (!options.ignoreUnknownFields) {
            throw new Error(`cannot decode message ${type.typeName} from JSON: key "${jsonKey}" is unknown`);
          }
          continue;
        }
        let localName2 = field.localName;
        let target = message;
        if (field.oneof) {
          if (jsonValue === null && field.kind == "scalar") {
            continue;
          }
          const seen = oneofSeen[field.oneof.localName];
          if (seen) {
            throw new Error(`cannot decode message ${type.typeName} from JSON: multiple keys for oneof "${field.oneof.name}" present: "${seen}", "${jsonKey}"`);
          }
          oneofSeen[field.oneof.localName] = jsonKey;
          target = target[field.oneof.localName] = { case: localName2 };
          localName2 = "value";
        }
        if (field.repeated) {
          if (jsonValue === null) {
            continue;
          }
          if (!Array.isArray(jsonValue)) {
            throw new Error(`cannot decode field ${type.typeName}.${field.name} from JSON: ${this.debug(jsonValue)}`);
          }
          const targetArray = target[localName2];
          for (const jsonItem of jsonValue) {
            if (jsonItem === null) {
              throw new Error(`cannot decode field ${type.typeName}.${field.name} from JSON: ${this.debug(jsonItem)}`);
            }
            let val;
            switch (field.kind) {
              case "message":
                val = field.T.fromJson(jsonItem, options);
                break;
              case "enum":
                val = readEnum(field.T, jsonItem, options.ignoreUnknownFields);
                if (val === void 0)
                  continue;
                break;
              case "scalar":
                try {
                  val = readScalar2(field.T, jsonItem);
                } catch (e) {
                  let m = `cannot decode field ${type.typeName}.${field.name} from JSON: ${this.debug(jsonItem)}`;
                  if (e instanceof Error && e.message.length > 0) {
                    m += `: ${e.message}`;
                  }
                  throw new Error(m);
                }
                break;
            }
            targetArray.push(val);
          }
        } else if (field.kind == "map") {
          if (jsonValue === null) {
            continue;
          }
          if (Array.isArray(jsonValue) || typeof jsonValue != "object") {
            throw new Error(`cannot decode field ${type.typeName}.${field.name} from JSON: ${this.debug(jsonValue)}`);
          }
          const targetMap = target[localName2];
          for (const [jsonMapKey, jsonMapValue] of Object.entries(jsonValue)) {
            if (jsonMapValue === null) {
              throw new Error(`cannot decode field ${type.typeName}.${field.name} from JSON: map value null`);
            }
            let val;
            switch (field.V.kind) {
              case "message":
                val = field.V.T.fromJson(jsonMapValue, options);
                break;
              case "enum":
                val = readEnum(field.V.T, jsonMapValue, options.ignoreUnknownFields);
                if (val === void 0)
                  continue;
                break;
              case "scalar":
                try {
                  val = readScalar2(field.V.T, jsonMapValue);
                } catch (e) {
                  let m = `cannot decode map value for field ${type.typeName}.${field.name} from JSON: ${this.debug(jsonValue)}`;
                  if (e instanceof Error && e.message.length > 0) {
                    m += `: ${e.message}`;
                  }
                  throw new Error(m);
                }
                break;
            }
            try {
              targetMap[readScalar2(field.K, field.K == ScalarType.BOOL ? jsonMapKey == "true" ? true : jsonMapKey == "false" ? false : jsonMapKey : jsonMapKey).toString()] = val;
            } catch (e) {
              let m = `cannot decode map key for field ${type.typeName}.${field.name} from JSON: ${this.debug(jsonValue)}`;
              if (e instanceof Error && e.message.length > 0) {
                m += `: ${e.message}`;
              }
              throw new Error(m);
            }
          }
        } else {
          switch (field.kind) {
            case "message":
              const messageType = field.T;
              if (jsonValue === null && messageType.typeName != "google.protobuf.Value") {
                if (field.oneof) {
                  throw new Error(`cannot decode field ${type.typeName}.${field.name} from JSON: null is invalid for oneof field "${jsonKey}"`);
                }
                continue;
              }
              if (target[localName2] instanceof Message) {
                target[localName2].fromJson(jsonValue, options);
              } else {
                target[localName2] = messageType.fromJson(jsonValue, options);
                if (messageType.fieldWrapper && !field.oneof) {
                  target[localName2] = messageType.fieldWrapper.unwrapField(target[localName2]);
                }
              }
              break;
            case "enum":
              const enumValue = readEnum(field.T, jsonValue, options.ignoreUnknownFields);
              if (enumValue !== void 0) {
                target[localName2] = enumValue;
              }
              break;
            case "scalar":
              try {
                target[localName2] = readScalar2(field.T, jsonValue);
              } catch (e) {
                let m = `cannot decode field ${type.typeName}.${field.name} from JSON: ${this.debug(jsonValue)}`;
                if (e instanceof Error && e.message.length > 0) {
                  m += `: ${e.message}`;
                }
                throw new Error(m);
              }
              break;
          }
        }
      }
      return message;
    },
    writeMessage(message, options) {
      const type = message.getType();
      const json = {};
      let field;
      try {
        for (const member of type.fields.byMember()) {
          let jsonValue;
          if (member.kind == "oneof") {
            const oneof = message[member.localName];
            if (oneof.value === void 0) {
              continue;
            }
            field = member.findField(oneof.case);
            if (!field) {
              throw "oneof case not found: " + oneof.case;
            }
            jsonValue = writeField(field, oneof.value, options);
          } else {
            field = member;
            jsonValue = writeField(field, message[field.localName], options);
          }
          if (jsonValue !== void 0) {
            json[options.useProtoFieldName ? field.name : field.jsonName] = jsonValue;
          }
        }
      } catch (e) {
        const m = field ? `cannot encode field ${type.typeName}.${field.name} to JSON` : `cannot encode message ${type.typeName} to JSON`;
        const r = e instanceof Error ? e.message : String(e);
        throw new Error(m + (r.length > 0 ? `: ${r}` : ""));
      }
      return json;
    },
    readScalar: readScalar2,
    writeScalar: writeScalar2,
    debug: debugJsonValue
  };
}
function debugJsonValue(json) {
  if (json === null) {
    return "null";
  }
  switch (typeof json) {
    case "object":
      return Array.isArray(json) ? "array" : "object";
    case "string":
      return json.length > 100 ? "string" : `"${json.split('"').join('\\"')}"`;
    default:
      return json.toString();
  }
}
function readScalar2(type, json) {
  switch (type) {
    case ScalarType.DOUBLE:
    case ScalarType.FLOAT:
      if (json === null)
        return 0;
      if (json === "NaN")
        return Number.NaN;
      if (json === "Infinity")
        return Number.POSITIVE_INFINITY;
      if (json === "-Infinity")
        return Number.NEGATIVE_INFINITY;
      if (json === "") {
        break;
      }
      if (typeof json == "string" && json.trim().length !== json.length) {
        break;
      }
      if (typeof json != "string" && typeof json != "number") {
        break;
      }
      const float = Number(json);
      if (Number.isNaN(float)) {
        break;
      }
      if (!Number.isFinite(float)) {
        break;
      }
      if (type == ScalarType.FLOAT)
        assertFloat32(float);
      return float;
    case ScalarType.INT32:
    case ScalarType.FIXED32:
    case ScalarType.SFIXED32:
    case ScalarType.SINT32:
    case ScalarType.UINT32:
      if (json === null)
        return 0;
      let int32;
      if (typeof json == "number")
        int32 = json;
      else if (typeof json == "string" && json.length > 0) {
        if (json.trim().length === json.length)
          int32 = Number(json);
      }
      if (int32 === void 0)
        break;
      if (type == ScalarType.UINT32)
        assertUInt32(int32);
      else
        assertInt32(int32);
      return int32;
    case ScalarType.INT64:
    case ScalarType.SFIXED64:
    case ScalarType.SINT64:
      if (json === null)
        return protoInt64.zero;
      if (typeof json != "number" && typeof json != "string")
        break;
      return protoInt64.parse(json);
    case ScalarType.FIXED64:
    case ScalarType.UINT64:
      if (json === null)
        return protoInt64.zero;
      if (typeof json != "number" && typeof json != "string")
        break;
      return protoInt64.uParse(json);
    case ScalarType.BOOL:
      if (json === null)
        return false;
      if (typeof json !== "boolean")
        break;
      return json;
    case ScalarType.STRING:
      if (json === null)
        return "";
      if (typeof json !== "string") {
        break;
      }
      try {
        encodeURIComponent(json);
      } catch (e) {
        throw new Error("invalid UTF8");
      }
      return json;
    case ScalarType.BYTES:
      if (json === null || json === "")
        return new Uint8Array(0);
      if (typeof json !== "string")
        break;
      return protoBase64.dec(json);
  }
  throw new Error();
}
function readEnum(type, json, ignoreUnknownFields) {
  if (json === null) {
    return 0;
  }
  switch (typeof json) {
    case "number":
      if (Number.isInteger(json)) {
        return json;
      }
      break;
    case "string":
      const value = type.findName(json);
      if (value || ignoreUnknownFields) {
        return value === null || value === void 0 ? void 0 : value.no;
      }
      break;
  }
  throw new Error(`cannot decode enum ${type.typeName} from JSON: ${debugJsonValue(json)}`);
}
function writeEnum(type, value, emitIntrinsicDefault, enumAsInteger) {
  var _a;
  if (value === void 0) {
    return value;
  }
  if (value === 0 && !emitIntrinsicDefault) {
    return void 0;
  }
  if (enumAsInteger) {
    return value;
  }
  if (type.typeName == "google.protobuf.NullValue") {
    return null;
  }
  const val = type.findNumber(value);
  return (_a = val === null || val === void 0 ? void 0 : val.name) !== null && _a !== void 0 ? _a : value;
}
function writeScalar2(type, value, emitIntrinsicDefault) {
  if (value === void 0) {
    return void 0;
  }
  switch (type) {
    case ScalarType.INT32:
    case ScalarType.SFIXED32:
    case ScalarType.SINT32:
    case ScalarType.FIXED32:
    case ScalarType.UINT32:
      assert(typeof value == "number");
      return value != 0 || emitIntrinsicDefault ? value : void 0;
    case ScalarType.FLOAT:
    case ScalarType.DOUBLE:
      assert(typeof value == "number");
      if (Number.isNaN(value))
        return "NaN";
      if (value === Number.POSITIVE_INFINITY)
        return "Infinity";
      if (value === Number.NEGATIVE_INFINITY)
        return "-Infinity";
      return value !== 0 || emitIntrinsicDefault ? value : void 0;
    case ScalarType.STRING:
      assert(typeof value == "string");
      return value.length > 0 || emitIntrinsicDefault ? value : void 0;
    case ScalarType.BOOL:
      assert(typeof value == "boolean");
      return value || emitIntrinsicDefault ? value : void 0;
    case ScalarType.UINT64:
    case ScalarType.FIXED64:
    case ScalarType.INT64:
    case ScalarType.SFIXED64:
    case ScalarType.SINT64:
      assert(typeof value == "bigint" || typeof value == "string" || typeof value == "number");
      return emitIntrinsicDefault || value != 0 ? value.toString(10) : void 0;
    case ScalarType.BYTES:
      assert(value instanceof Uint8Array);
      return emitIntrinsicDefault || value.byteLength > 0 ? protoBase64.enc(value) : void 0;
  }
}

// node_modules/@bufbuild/protobuf/dist/esm/private/json-format-proto3.js
function makeJsonFormatProto3() {
  return makeJsonFormatCommon((writeEnum2, writeScalar3) => {
    return function writeField(field, value, options) {
      if (field.kind == "map") {
        const jsonObj = {};
        switch (field.V.kind) {
          case "scalar":
            for (const [entryKey, entryValue] of Object.entries(value)) {
              const val = writeScalar3(field.V.T, entryValue, true);
              assert(val !== void 0);
              jsonObj[entryKey.toString()] = val;
            }
            break;
          case "message":
            for (const [entryKey, entryValue] of Object.entries(value)) {
              jsonObj[entryKey.toString()] = entryValue.toJson(options);
            }
            break;
          case "enum":
            const enumType = field.V.T;
            for (const [entryKey, entryValue] of Object.entries(value)) {
              assert(entryValue === void 0 || typeof entryValue == "number");
              const val = writeEnum2(enumType, entryValue, true, options.enumAsInteger);
              assert(val !== void 0);
              jsonObj[entryKey.toString()] = val;
            }
            break;
        }
        return options.emitDefaultValues || Object.keys(jsonObj).length > 0 ? jsonObj : void 0;
      } else if (field.repeated) {
        const jsonArr = [];
        switch (field.kind) {
          case "scalar":
            for (let i = 0; i < value.length; i++) {
              jsonArr.push(writeScalar3(field.T, value[i], true));
            }
            break;
          case "enum":
            for (let i = 0; i < value.length; i++) {
              jsonArr.push(writeEnum2(field.T, value[i], true, options.enumAsInteger));
            }
            break;
          case "message":
            for (let i = 0; i < value.length; i++) {
              jsonArr.push(wrapField(field.T, value[i]).toJson(options));
            }
            break;
        }
        return options.emitDefaultValues || jsonArr.length > 0 ? jsonArr : void 0;
      } else {
        switch (field.kind) {
          case "scalar":
            return writeScalar3(field.T, value, !!field.oneof || field.opt || options.emitDefaultValues);
          case "enum":
            return writeEnum2(field.T, value, !!field.oneof || field.opt || options.emitDefaultValues, options.enumAsInteger);
          case "message":
            return value !== void 0 ? wrapField(field.T, value).toJson(options) : void 0;
        }
      }
    };
  });
}

// node_modules/@bufbuild/protobuf/dist/esm/private/util-common.js
function makeUtilCommon() {
  return {
    setEnumType,
    initPartial(source, target) {
      if (source === void 0) {
        return;
      }
      const type = target.getType();
      for (const member of type.fields.byMember()) {
        const localName2 = member.localName, t = target, s = source;
        if (s[localName2] === void 0) {
          continue;
        }
        switch (member.kind) {
          case "oneof":
            const sk = s[localName2].case;
            if (sk === void 0) {
              continue;
            }
            const sourceField = member.findField(sk);
            let val = s[localName2].value;
            if (sourceField && sourceField.kind == "message" && !(val instanceof sourceField.T)) {
              val = new sourceField.T(val);
            }
            t[localName2] = { case: sk, value: val };
            break;
          case "scalar":
          case "enum":
            t[localName2] = s[localName2];
            break;
          case "map":
            switch (member.V.kind) {
              case "scalar":
              case "enum":
                Object.assign(t[localName2], s[localName2]);
                break;
              case "message":
                const messageType = member.V.T;
                for (const k of Object.keys(s[localName2])) {
                  let val2 = s[localName2][k];
                  if (!messageType.fieldWrapper) {
                    val2 = new messageType(val2);
                  }
                  t[localName2][k] = val2;
                }
                break;
            }
            break;
          case "message":
            const mt = member.T;
            if (member.repeated) {
              t[localName2] = s[localName2].map((val2) => val2 instanceof mt ? val2 : new mt(val2));
            } else if (s[localName2] !== void 0) {
              const val2 = s[localName2];
              if (mt.fieldWrapper) {
                t[localName2] = val2;
              } else {
                t[localName2] = val2 instanceof mt ? val2 : new mt(val2);
              }
            }
            break;
        }
      }
    },
    equals(type, a, b) {
      if (a === b) {
        return true;
      }
      if (!a || !b) {
        return false;
      }
      return type.fields.byMember().every((m) => {
        const va = a[m.localName];
        const vb = b[m.localName];
        if (m.repeated) {
          if (va.length !== vb.length) {
            return false;
          }
          switch (m.kind) {
            case "message":
              return va.every((a2, i) => m.T.equals(a2, vb[i]));
            case "scalar":
              return va.every((a2, i) => scalarEquals(m.T, a2, vb[i]));
            case "enum":
              return va.every((a2, i) => scalarEquals(ScalarType.INT32, a2, vb[i]));
          }
          throw new Error(`repeated cannot contain ${m.kind}`);
        }
        switch (m.kind) {
          case "message":
            return m.T.equals(va, vb);
          case "enum":
            return scalarEquals(ScalarType.INT32, va, vb);
          case "scalar":
            return scalarEquals(m.T, va, vb);
          case "oneof":
            if (va.case !== vb.case) {
              return false;
            }
            const s = m.findField(va.case);
            if (s === void 0) {
              return true;
            }
            switch (s.kind) {
              case "message":
                return s.T.equals(va.value, vb.value);
              case "enum":
                return scalarEquals(ScalarType.INT32, va.value, vb.value);
              case "scalar":
                return scalarEquals(s.T, va.value, vb.value);
            }
            throw new Error(`oneof cannot contain ${s.kind}`);
          case "map":
            const keys = Object.keys(va).concat(Object.keys(vb));
            switch (m.V.kind) {
              case "message":
                const messageType = m.V.T;
                return keys.every((k) => messageType.equals(va[k], vb[k]));
              case "enum":
                return keys.every((k) => scalarEquals(ScalarType.INT32, va[k], vb[k]));
              case "scalar":
                const scalarType = m.V.T;
                return keys.every((k) => scalarEquals(scalarType, va[k], vb[k]));
            }
            break;
        }
      });
    },
    clone(message) {
      const type = message.getType(), target = new type(), any = target;
      for (const member of type.fields.byMember()) {
        const source = message[member.localName];
        let copy;
        if (member.repeated) {
          copy = source.map((e) => cloneSingularField(member, e));
        } else if (member.kind == "map") {
          copy = any[member.localName];
          for (const [key, v] of Object.entries(source)) {
            copy[key] = cloneSingularField(member.V, v);
          }
        } else if (member.kind == "oneof") {
          const f = member.findField(source.case);
          copy = f ? { case: source.case, value: cloneSingularField(f, source.value) } : { case: void 0 };
        } else {
          copy = cloneSingularField(member, source);
        }
        any[member.localName] = copy;
      }
      return target;
    }
  };
}
function cloneSingularField(field, value) {
  if (value === void 0) {
    return value;
  }
  if (value instanceof Message) {
    return value.clone();
  }
  if (value instanceof Uint8Array) {
    const c = new Uint8Array(value.byteLength);
    c.set(value);
    return c;
  }
  return value;
}

// node_modules/@bufbuild/protobuf/dist/esm/private/field-list.js
var InternalFieldList = class {
  constructor(fields, normalizer) {
    this._fields = fields;
    this._normalizer = normalizer;
  }
  findJsonName(jsonName) {
    if (!this.jsonNames) {
      const t = {};
      for (const f of this.list()) {
        t[f.jsonName] = t[f.name] = f;
      }
      this.jsonNames = t;
    }
    return this.jsonNames[jsonName];
  }
  find(fieldNo) {
    if (!this.numbers) {
      const t = {};
      for (const f of this.list()) {
        t[f.no] = f;
      }
      this.numbers = t;
    }
    return this.numbers[fieldNo];
  }
  list() {
    if (!this.all) {
      this.all = this._normalizer(this._fields);
    }
    return this.all;
  }
  byNumber() {
    if (!this.numbersAsc) {
      this.numbersAsc = this.list().concat().sort((a, b) => a.no - b.no);
    }
    return this.numbersAsc;
  }
  byMember() {
    if (!this.members) {
      this.members = [];
      const a = this.members;
      let o;
      for (const f of this.list()) {
        if (f.oneof) {
          if (f.oneof !== o) {
            o = f.oneof;
            a.push(o);
          }
        } else {
          a.push(f);
        }
      }
    }
    return this.members;
  }
};

// node_modules/@bufbuild/protobuf/dist/esm/private/names.js
function localName(desc) {
  switch (desc.kind) {
    case "field":
      return localFieldName(desc.name, desc.oneof !== void 0);
    case "oneof":
      return localOneofName(desc.name);
    case "enum":
    case "message":
    case "service": {
      const pkg = desc.file.proto.package;
      const offset = pkg === void 0 ? 0 : pkg.length + 1;
      const name = desc.typeName.substring(offset).replace(/\./g, "_");
      return safeObjectProperty(safeIdentifier(name));
    }
    case "enum_value": {
      const sharedPrefix = desc.parent.sharedPrefix;
      if (sharedPrefix === void 0) {
        return desc.name;
      }
      const name = desc.name.substring(sharedPrefix.length);
      return safeObjectProperty(name);
    }
    case "rpc": {
      let name = desc.name;
      if (name.length == 0) {
        return name;
      }
      name = name[0].toLowerCase() + name.substring(1);
      return safeObjectProperty(name);
    }
  }
}
function localFieldName(protoName, inOneof) {
  const name = protoCamelCase(protoName);
  if (inOneof) {
    return name;
  }
  return safeObjectProperty(safeMessageProperty(name));
}
function localOneofName(protoName) {
  return localFieldName(protoName, false);
}
var fieldJsonName = protoCamelCase;
function findEnumSharedPrefix(enumName, valueNames) {
  const prefix = camelToSnakeCase(enumName) + "_";
  for (const name of valueNames) {
    if (!name.toLowerCase().startsWith(prefix)) {
      return void 0;
    }
    const shortName = name.substring(prefix.length);
    if (shortName.length == 0) {
      return void 0;
    }
    if (/^\d/.test(shortName)) {
      return void 0;
    }
  }
  return prefix;
}
function camelToSnakeCase(camel) {
  return (camel.substring(0, 1) + camel.substring(1).replace(/[A-Z]/g, (c) => "_" + c)).toLowerCase();
}
function protoCamelCase(snakeCase) {
  let capNext = false;
  const b = [];
  for (let i = 0; i < snakeCase.length; i++) {
    let c = snakeCase.charAt(i);
    switch (c) {
      case "_":
        capNext = true;
        break;
      case "0":
      case "1":
      case "2":
      case "3":
      case "4":
      case "5":
      case "6":
      case "7":
      case "8":
      case "9":
        b.push(c);
        capNext = false;
        break;
      default:
        if (capNext) {
          capNext = false;
          c = c.toUpperCase();
        }
        b.push(c);
        break;
    }
  }
  return b.join("");
}
var reservedIdentifiers = /* @__PURE__ */ new Set([
  // ECMAScript 2015 keywords
  "break",
  "case",
  "catch",
  "class",
  "const",
  "continue",
  "debugger",
  "default",
  "delete",
  "do",
  "else",
  "export",
  "extends",
  "false",
  "finally",
  "for",
  "function",
  "if",
  "import",
  "in",
  "instanceof",
  "new",
  "null",
  "return",
  "super",
  "switch",
  "this",
  "throw",
  "true",
  "try",
  "typeof",
  "var",
  "void",
  "while",
  "with",
  "yield",
  // ECMAScript 2015 future reserved keywords
  "enum",
  "implements",
  "interface",
  "let",
  "package",
  "private",
  "protected",
  "public",
  "static",
  // Class name cannot be 'Object' when targeting ES5 with module CommonJS
  "Object",
  // TypeScript keywords that cannot be used for types (as opposed to variables)
  "bigint",
  "number",
  "boolean",
  "string",
  "object",
  // Identifiers reserved for the runtime, so we can generate legible code
  "globalThis",
  "Uint8Array",
  "Partial"
]);
var reservedObjectProperties = /* @__PURE__ */ new Set([
  // names reserved by JavaScript
  "constructor",
  "toString",
  "toJSON",
  "valueOf"
]);
var reservedMessageProperties = /* @__PURE__ */ new Set([
  // names reserved by the runtime
  "getType",
  "clone",
  "equals",
  "fromBinary",
  "fromJson",
  "fromJsonString",
  "toBinary",
  "toJson",
  "toJsonString",
  // names reserved by the runtime for the future
  "toObject"
]);
var fallback = (name) => `${name}$`;
var safeMessageProperty = (name) => {
  if (reservedMessageProperties.has(name)) {
    return fallback(name);
  }
  return name;
};
var safeObjectProperty = (name) => {
  if (reservedObjectProperties.has(name)) {
    return fallback(name);
  }
  return name;
};
var safeIdentifier = (name) => {
  if (reservedIdentifiers.has(name)) {
    return fallback(name);
  }
  return name;
};

// node_modules/@bufbuild/protobuf/dist/esm/private/field.js
var InternalOneofInfo = class {
  constructor(name) {
    this.kind = "oneof";
    this.repeated = false;
    this.packed = false;
    this.opt = false;
    this.default = void 0;
    this.fields = [];
    this.name = name;
    this.localName = localOneofName(name);
  }
  addField(field) {
    assert(field.oneof === this, `field ${field.name} not one of ${this.name}`);
    this.fields.push(field);
  }
  findField(localName2) {
    if (!this._lookup) {
      this._lookup = /* @__PURE__ */ Object.create(null);
      for (let i = 0; i < this.fields.length; i++) {
        this._lookup[this.fields[i].localName] = this.fields[i];
      }
    }
    return this._lookup[localName2];
  }
};

// node_modules/@bufbuild/protobuf/dist/esm/proto3.js
var proto3 = makeProtoRuntime("proto3", makeJsonFormatProto3(), makeBinaryFormatProto3(), Object.assign(Object.assign({}, makeUtilCommon()), {
  newFieldList(fields) {
    return new InternalFieldList(fields, normalizeFieldInfosProto3);
  },
  initFields(target) {
    for (const member of target.getType().fields.byMember()) {
      if (member.opt) {
        continue;
      }
      const name = member.localName, t = target;
      if (member.repeated) {
        t[name] = [];
        continue;
      }
      switch (member.kind) {
        case "oneof":
          t[name] = { case: void 0 };
          break;
        case "enum":
          t[name] = 0;
          break;
        case "map":
          t[name] = {};
          break;
        case "scalar":
          t[name] = scalarDefaultValue(member.T);
          break;
        case "message":
          break;
      }
    }
  }
}));
function normalizeFieldInfosProto3(fieldInfos) {
  var _a, _b, _c;
  const r = [];
  let o;
  for (const field of typeof fieldInfos == "function" ? fieldInfos() : fieldInfos) {
    const f = field;
    f.localName = localFieldName(field.name, field.oneof !== void 0);
    f.jsonName = (_a = field.jsonName) !== null && _a !== void 0 ? _a : fieldJsonName(field.name);
    f.repeated = (_b = field.repeated) !== null && _b !== void 0 ? _b : false;
    f.packed = (_c = field.packed) !== null && _c !== void 0 ? _c : field.kind == "enum" || field.kind == "scalar" && field.T != ScalarType.BYTES && field.T != ScalarType.STRING;
    if (field.oneof !== void 0) {
      const ooname = typeof field.oneof == "string" ? field.oneof : field.oneof.name;
      if (!o || o.name != ooname) {
        o = new InternalOneofInfo(ooname);
      }
      f.oneof = o;
      o.addField(f);
    }
    r.push(f);
  }
  return r;
}

// node_modules/@bufbuild/protobuf/dist/esm/private/binary-format-proto2.js
function makeBinaryFormatProto2() {
  return Object.assign(Object.assign({}, makeBinaryFormatCommon()), { writeMessage(message, writer, options) {
    const type = message.getType();
    let field;
    try {
      for (field of type.fields.byNumber()) {
        let value, repeated = field.repeated, localName2 = field.localName;
        if (field.oneof) {
          const oneof = message[field.oneof.localName];
          if (oneof.case !== localName2) {
            continue;
          }
          value = oneof.value;
        } else {
          value = message[localName2];
          if (value === void 0 && !field.oneof && !field.opt) {
            throw new Error(`cannot encode field ${type.typeName}.${field.name} to binary: required field not set`);
          }
        }
        switch (field.kind) {
          case "scalar":
          case "enum":
            let scalarType = field.kind == "enum" ? ScalarType.INT32 : field.T;
            if (repeated) {
              if (field.packed) {
                writePacked(writer, scalarType, field.no, value);
              } else {
                for (const item of value) {
                  writeScalar(writer, scalarType, field.no, item, true);
                }
              }
            } else {
              if (value !== void 0) {
                writeScalar(writer, scalarType, field.no, value, true);
              }
            }
            break;
          case "message":
            if (repeated) {
              for (const item of value) {
                writeMessageField(writer, options, field.T, field.no, item);
              }
            } else {
              writeMessageField(writer, options, field.T, field.no, value);
            }
            break;
          case "map":
            for (const [key, val] of Object.entries(value)) {
              writeMapEntry(writer, options, field, key, val);
            }
            break;
        }
      }
    } catch (e) {
      let m = field ? `cannot encode field ${type.typeName}.${field === null || field === void 0 ? void 0 : field.name} to binary` : `cannot encode message ${type.typeName} to binary`;
      let r = e instanceof Error ? e.message : String(e);
      throw new Error(m + (r.length > 0 ? `: ${r}` : ""));
    }
    if (options.writeUnknownFields) {
      this.writeUnknownFields(message, writer);
    }
    return writer;
  } });
}

// node_modules/@bufbuild/protobuf/dist/esm/private/json-format-proto2.js
function makeJsonFormatProto2() {
  return makeJsonFormatCommon((writeEnum2, writeScalar3) => {
    return function writeField(field, value, options) {
      if (field.kind == "map") {
        const jsonObj = {};
        switch (field.V.kind) {
          case "scalar":
            for (const [entryKey, entryValue] of Object.entries(value)) {
              const val = writeScalar3(field.V.T, entryValue, true);
              assert(val !== void 0);
              jsonObj[entryKey.toString()] = val;
            }
            break;
          case "message":
            for (const [entryKey, entryValue] of Object.entries(value)) {
              jsonObj[entryKey.toString()] = entryValue.toJson(options);
            }
            break;
          case "enum":
            const enumType = field.V.T;
            for (const [entryKey, entryValue] of Object.entries(value)) {
              assert(entryValue === void 0 || typeof entryValue == "number");
              const val = writeEnum2(enumType, entryValue, true, options.enumAsInteger);
              assert(val !== void 0);
              jsonObj[entryKey.toString()] = val;
            }
            break;
        }
        return options.emitDefaultValues || Object.keys(jsonObj).length > 0 ? jsonObj : void 0;
      } else if (field.repeated) {
        const jsonArr = [];
        switch (field.kind) {
          case "scalar":
            for (let i = 0; i < value.length; i++) {
              jsonArr.push(writeScalar3(field.T, value[i], true));
            }
            break;
          case "enum":
            for (let i = 0; i < value.length; i++) {
              jsonArr.push(writeEnum2(field.T, value[i], true, options.enumAsInteger));
            }
            break;
          case "message":
            for (let i = 0; i < value.length; i++) {
              jsonArr.push(value[i].toJson(options));
            }
            break;
        }
        return options.emitDefaultValues || jsonArr.length > 0 ? jsonArr : void 0;
      } else {
        if (value === void 0) {
          if (!field.oneof && !field.opt) {
            throw `required field not set`;
          }
          return void 0;
        }
        switch (field.kind) {
          case "scalar":
            return writeScalar3(field.T, value, true);
          case "enum":
            return writeEnum2(field.T, value, true, options.enumAsInteger);
          case "message":
            return wrapField(field.T, value).toJson(options);
        }
      }
    };
  });
}

// node_modules/@bufbuild/protobuf/dist/esm/proto2.js
var proto2 = makeProtoRuntime("proto2", makeJsonFormatProto2(), makeBinaryFormatProto2(), Object.assign(Object.assign({}, makeUtilCommon()), {
  newFieldList(fields) {
    return new InternalFieldList(fields, normalizeFieldInfosProto2);
  },
  initFields(target) {
    for (const member of target.getType().fields.byMember()) {
      const name = member.localName, t = target;
      if (member.repeated) {
        t[name] = [];
        continue;
      }
      switch (member.kind) {
        case "oneof":
          t[name] = { case: void 0 };
          break;
        case "map":
          t[name] = {};
          break;
        case "scalar":
        case "enum":
        case "message":
          break;
      }
    }
  }
}));
function normalizeFieldInfosProto2(fieldInfos) {
  var _a, _b, _c;
  const r = [];
  let o;
  for (const field of typeof fieldInfos == "function" ? fieldInfos() : fieldInfos) {
    const f = field;
    f.localName = localFieldName(field.name, field.oneof !== void 0);
    f.jsonName = (_a = field.jsonName) !== null && _a !== void 0 ? _a : fieldJsonName(field.name);
    f.repeated = (_b = field.repeated) !== null && _b !== void 0 ? _b : false;
    f.packed = (_c = field.packed) !== null && _c !== void 0 ? _c : false;
    if (field.oneof !== void 0) {
      const ooname = typeof field.oneof == "string" ? field.oneof : field.oneof.name;
      if (!o || o.name != ooname) {
        o = new InternalOneofInfo(ooname);
      }
      f.oneof = o;
      o.addField(f);
    }
    r.push(f);
  }
  return r;
}

// node_modules/@bufbuild/protobuf/dist/esm/proto-double.js
var protoDouble = {
  NaN: Number.NaN,
  POSITIVE_INFINITY: Number.POSITIVE_INFINITY,
  NEGATIVE_INFINITY: Number.NEGATIVE_INFINITY
};

// node_modules/@bufbuild/protobuf/dist/esm/proto-delimited.js
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
var protoDelimited = {
  /**
   * Serialize a message, prefixing it with its size.
   */
  enc(message, options) {
    const opt = makeBinaryFormatCommon().makeWriteOptions(options);
    return opt.writerFactory().bytes(message.toBinary(opt)).finish();
  },
  /**
   * Parse a size-delimited message, ignoring extra bytes.
   */
  dec(type, bytes, options) {
    const opt = makeBinaryFormatCommon().makeReadOptions(options);
    return type.fromBinary(opt.readerFactory(bytes).bytes(), opt);
  },
  /**
   * Parse a stream of size-delimited messages.
   */
  decStream(type, iterable) {
    return __asyncGenerator(this, arguments, function* decStream_1() {
      var _a, e_1, _b, _c;
      function append(buffer2, chunk) {
        const n = new Uint8Array(buffer2.byteLength + chunk.byteLength);
        n.set(buffer2);
        n.set(chunk, buffer2.length);
        return n;
      }
      let buffer = new Uint8Array(0);
      try {
        for (var _d = true, iterable_1 = __asyncValues(iterable), iterable_1_1; iterable_1_1 = yield __await(iterable_1.next()), _a = iterable_1_1.done, !_a; _d = true) {
          _c = iterable_1_1.value;
          _d = false;
          const chunk = _c;
          buffer = append(buffer, chunk);
          for (; ; ) {
            const size = protoDelimited.peekSize(buffer);
            if (size.eof) {
              break;
            }
            if (size.offset + size.size > buffer.byteLength) {
              break;
            }
            yield yield __await(protoDelimited.dec(type, buffer));
            buffer = buffer.subarray(size.offset + size.size);
          }
        }
      } catch (e_1_1) {
        e_1 = { error: e_1_1 };
      } finally {
        try {
          if (!_d && !_a && (_b = iterable_1.return))
            yield __await(_b.call(iterable_1));
        } finally {
          if (e_1)
            throw e_1.error;
        }
      }
      if (buffer.byteLength > 0) {
        throw new Error("incomplete data");
      }
    });
  },
  /**
   * Decodes the size from the given size-delimited message, which may be
   * incomplete.
   *
   * Returns an object with the following properties:
   * - size: The size of the delimited message in bytes
   * - offset: The offset in the given byte array where the message starts
   * - eof: true
   *
   * If the size-delimited data does not include all bytes of the varint size,
   * the following object is returned:
   * - size: null
   * - offset: null
   * - eof: false
   *
   * This function can be used to implement parsing of size-delimited messages
   * from a stream.
   */
  peekSize(data) {
    const sizeEof = { eof: true, size: null, offset: null };
    for (let i = 0; i < 10; i++) {
      if (i > data.byteLength) {
        return sizeEof;
      }
      if ((data[i] & 128) == 0) {
        const reader = new BinaryReader(data);
        let size;
        try {
          size = reader.uint32();
        } catch (e) {
          if (e instanceof RangeError) {
            return sizeEof;
          }
          throw e;
        }
        return {
          eof: false,
          size,
          offset: reader.pos
        };
      }
    }
    throw new Error("invalid varint");
  }
};

// node_modules/@bufbuild/protobuf/dist/esm/private/reify-wkt.js
function reifyWkt(message) {
  switch (message.typeName) {
    case "google.protobuf.Any": {
      const typeUrl = message.fields.find((f) => f.number == 1 && f.fieldKind == "scalar" && f.scalar === ScalarType.STRING);
      const value = message.fields.find((f) => f.number == 2 && f.fieldKind == "scalar" && f.scalar === ScalarType.BYTES);
      if (typeUrl && value) {
        return {
          typeName: message.typeName,
          typeUrl,
          value
        };
      }
      break;
    }
    case "google.protobuf.Timestamp": {
      const seconds = message.fields.find((f) => f.number == 1 && f.fieldKind == "scalar" && f.scalar === ScalarType.INT64);
      const nanos = message.fields.find((f) => f.number == 2 && f.fieldKind == "scalar" && f.scalar === ScalarType.INT32);
      if (seconds && nanos) {
        return {
          typeName: message.typeName,
          seconds,
          nanos
        };
      }
      break;
    }
    case "google.protobuf.Duration": {
      const seconds = message.fields.find((f) => f.number == 1 && f.fieldKind == "scalar" && f.scalar === ScalarType.INT64);
      const nanos = message.fields.find((f) => f.number == 2 && f.fieldKind == "scalar" && f.scalar === ScalarType.INT32);
      if (seconds && nanos) {
        return {
          typeName: message.typeName,
          seconds,
          nanos
        };
      }
      break;
    }
    case "google.protobuf.Struct": {
      const fields = message.fields.find((f) => f.number == 1 && !f.repeated);
      if ((fields === null || fields === void 0 ? void 0 : fields.fieldKind) !== "map" || fields.mapValue.kind !== "message" || fields.mapValue.message.typeName !== "google.protobuf.Value") {
        break;
      }
      return { typeName: message.typeName, fields };
    }
    case "google.protobuf.Value": {
      const kind = message.oneofs.find((o) => o.name === "kind");
      const nullValue = message.fields.find((f) => f.number == 1 && f.oneof === kind);
      if ((nullValue === null || nullValue === void 0 ? void 0 : nullValue.fieldKind) !== "enum" || nullValue.enum.typeName !== "google.protobuf.NullValue") {
        return void 0;
      }
      const numberValue = message.fields.find((f) => f.number == 2 && f.fieldKind == "scalar" && f.scalar === ScalarType.DOUBLE && f.oneof === kind);
      const stringValue = message.fields.find((f) => f.number == 3 && f.fieldKind == "scalar" && f.scalar === ScalarType.STRING && f.oneof === kind);
      const boolValue = message.fields.find((f) => f.number == 4 && f.fieldKind == "scalar" && f.scalar === ScalarType.BOOL && f.oneof === kind);
      const structValue = message.fields.find((f) => f.number == 5 && f.oneof === kind);
      if ((structValue === null || structValue === void 0 ? void 0 : structValue.fieldKind) !== "message" || structValue.message.typeName !== "google.protobuf.Struct") {
        return void 0;
      }
      const listValue = message.fields.find((f) => f.number == 6 && f.oneof === kind);
      if ((listValue === null || listValue === void 0 ? void 0 : listValue.fieldKind) !== "message" || listValue.message.typeName !== "google.protobuf.ListValue") {
        return void 0;
      }
      if (kind && numberValue && stringValue && boolValue) {
        return {
          typeName: message.typeName,
          kind,
          nullValue,
          numberValue,
          stringValue,
          boolValue,
          structValue,
          listValue
        };
      }
      break;
    }
    case "google.protobuf.ListValue": {
      const values = message.fields.find((f) => f.number == 1 && f.repeated);
      if ((values === null || values === void 0 ? void 0 : values.fieldKind) != "message" || values.message.typeName !== "google.protobuf.Value") {
        break;
      }
      return { typeName: message.typeName, values };
    }
    case "google.protobuf.FieldMask": {
      const paths = message.fields.find((f) => f.number == 1 && f.fieldKind == "scalar" && f.scalar === ScalarType.STRING && f.repeated);
      if (paths) {
        return { typeName: message.typeName, paths };
      }
      break;
    }
    case "google.protobuf.DoubleValue":
    case "google.protobuf.FloatValue":
    case "google.protobuf.Int64Value":
    case "google.protobuf.UInt64Value":
    case "google.protobuf.Int32Value":
    case "google.protobuf.UInt32Value":
    case "google.protobuf.BoolValue":
    case "google.protobuf.StringValue":
    case "google.protobuf.BytesValue": {
      const value = message.fields.find((f) => f.number == 1 && f.name == "value");
      if (!value) {
        break;
      }
      if (value.fieldKind !== "scalar") {
        break;
      }
      return { typeName: message.typeName, value };
    }
  }
  return void 0;
}

// node_modules/@bufbuild/protobuf/dist/esm/codegen-info.js
var packageName = "@bufbuild/protobuf";
var codegenInfo = {
  packageName,
  localName,
  reifyWkt,
  getUnwrappedFieldType,
  scalarDefaultValue,
  safeIdentifier,
  safeObjectProperty,
  // prettier-ignore
  symbols: {
    proto2: { typeOnly: false, privateImportPath: "./proto2.js", publicImportPath: packageName },
    proto3: { typeOnly: false, privateImportPath: "./proto3.js", publicImportPath: packageName },
    Message: { typeOnly: false, privateImportPath: "./message.js", publicImportPath: packageName },
    PartialMessage: { typeOnly: true, privateImportPath: "./message.js", publicImportPath: packageName },
    PlainMessage: { typeOnly: true, privateImportPath: "./message.js", publicImportPath: packageName },
    FieldList: { typeOnly: true, privateImportPath: "./field-list.js", publicImportPath: packageName },
    MessageType: { typeOnly: true, privateImportPath: "./message-type.js", publicImportPath: packageName },
    BinaryReadOptions: { typeOnly: true, privateImportPath: "./binary-format.js", publicImportPath: packageName },
    BinaryWriteOptions: { typeOnly: true, privateImportPath: "./binary-format.js", publicImportPath: packageName },
    JsonReadOptions: { typeOnly: true, privateImportPath: "./json-format.js", publicImportPath: packageName },
    JsonWriteOptions: { typeOnly: true, privateImportPath: "./json-format.js", publicImportPath: packageName },
    JsonValue: { typeOnly: true, privateImportPath: "./json-format.js", publicImportPath: packageName },
    JsonObject: { typeOnly: true, privateImportPath: "./json-format.js", publicImportPath: packageName },
    protoDouble: { typeOnly: false, privateImportPath: "./proto-double.js", publicImportPath: packageName },
    protoInt64: { typeOnly: false, privateImportPath: "./proto-int64.js", publicImportPath: packageName },
    ScalarType: { typeOnly: false, privateImportPath: "./field.js", publicImportPath: packageName },
    MethodKind: { typeOnly: false, privateImportPath: "./service-type.js", publicImportPath: packageName },
    MethodIdempotency: { typeOnly: false, privateImportPath: "./service-type.js", publicImportPath: packageName },
    IMessageTypeRegistry: { typeOnly: true, privateImportPath: "./type-registry.js", publicImportPath: packageName }
  },
  wktSourceFiles: [
    "google/protobuf/compiler/plugin.proto",
    "google/protobuf/any.proto",
    "google/protobuf/api.proto",
    "google/protobuf/descriptor.proto",
    "google/protobuf/duration.proto",
    "google/protobuf/empty.proto",
    "google/protobuf/field_mask.proto",
    "google/protobuf/source_context.proto",
    "google/protobuf/struct.proto",
    "google/protobuf/timestamp.proto",
    "google/protobuf/type.proto",
    "google/protobuf/wrappers.proto"
  ]
};

// node_modules/@bufbuild/protobuf/dist/esm/service-type.js
var MethodKind;
(function(MethodKind2) {
  MethodKind2[MethodKind2["Unary"] = 0] = "Unary";
  MethodKind2[MethodKind2["ServerStreaming"] = 1] = "ServerStreaming";
  MethodKind2[MethodKind2["ClientStreaming"] = 2] = "ClientStreaming";
  MethodKind2[MethodKind2["BiDiStreaming"] = 3] = "BiDiStreaming";
})(MethodKind || (MethodKind = {}));
var MethodIdempotency;
(function(MethodIdempotency2) {
  MethodIdempotency2[MethodIdempotency2["NoSideEffects"] = 1] = "NoSideEffects";
  MethodIdempotency2[MethodIdempotency2["Idempotent"] = 2] = "Idempotent";
})(MethodIdempotency || (MethodIdempotency = {}));

// node_modules/@bufbuild/protobuf/dist/esm/google/protobuf/descriptor_pb.js
var FileDescriptorSet = class _FileDescriptorSet extends Message {
  constructor(data) {
    super();
    this.file = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _FileDescriptorSet().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _FileDescriptorSet().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _FileDescriptorSet().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_FileDescriptorSet, a, b);
  }
};
FileDescriptorSet.runtime = proto2;
FileDescriptorSet.typeName = "google.protobuf.FileDescriptorSet";
FileDescriptorSet.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "file", kind: "message", T: FileDescriptorProto, repeated: true }
]);
var FileDescriptorProto = class _FileDescriptorProto extends Message {
  constructor(data) {
    super();
    this.dependency = [];
    this.publicDependency = [];
    this.weakDependency = [];
    this.messageType = [];
    this.enumType = [];
    this.service = [];
    this.extension = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _FileDescriptorProto().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _FileDescriptorProto().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _FileDescriptorProto().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_FileDescriptorProto, a, b);
  }
};
FileDescriptorProto.runtime = proto2;
FileDescriptorProto.typeName = "google.protobuf.FileDescriptorProto";
FileDescriptorProto.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "name", kind: "scalar", T: 9, opt: true },
  { no: 2, name: "package", kind: "scalar", T: 9, opt: true },
  { no: 3, name: "dependency", kind: "scalar", T: 9, repeated: true },
  { no: 10, name: "public_dependency", kind: "scalar", T: 5, repeated: true },
  { no: 11, name: "weak_dependency", kind: "scalar", T: 5, repeated: true },
  { no: 4, name: "message_type", kind: "message", T: DescriptorProto, repeated: true },
  { no: 5, name: "enum_type", kind: "message", T: EnumDescriptorProto, repeated: true },
  { no: 6, name: "service", kind: "message", T: ServiceDescriptorProto, repeated: true },
  { no: 7, name: "extension", kind: "message", T: FieldDescriptorProto, repeated: true },
  { no: 8, name: "options", kind: "message", T: FileOptions, opt: true },
  { no: 9, name: "source_code_info", kind: "message", T: SourceCodeInfo, opt: true },
  { no: 12, name: "syntax", kind: "scalar", T: 9, opt: true },
  { no: 13, name: "edition", kind: "scalar", T: 9, opt: true }
]);
var DescriptorProto = class _DescriptorProto extends Message {
  constructor(data) {
    super();
    this.field = [];
    this.extension = [];
    this.nestedType = [];
    this.enumType = [];
    this.extensionRange = [];
    this.oneofDecl = [];
    this.reservedRange = [];
    this.reservedName = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _DescriptorProto().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _DescriptorProto().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _DescriptorProto().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_DescriptorProto, a, b);
  }
};
DescriptorProto.runtime = proto2;
DescriptorProto.typeName = "google.protobuf.DescriptorProto";
DescriptorProto.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "name", kind: "scalar", T: 9, opt: true },
  { no: 2, name: "field", kind: "message", T: FieldDescriptorProto, repeated: true },
  { no: 6, name: "extension", kind: "message", T: FieldDescriptorProto, repeated: true },
  { no: 3, name: "nested_type", kind: "message", T: DescriptorProto, repeated: true },
  { no: 4, name: "enum_type", kind: "message", T: EnumDescriptorProto, repeated: true },
  { no: 5, name: "extension_range", kind: "message", T: DescriptorProto_ExtensionRange, repeated: true },
  { no: 8, name: "oneof_decl", kind: "message", T: OneofDescriptorProto, repeated: true },
  { no: 7, name: "options", kind: "message", T: MessageOptions, opt: true },
  { no: 9, name: "reserved_range", kind: "message", T: DescriptorProto_ReservedRange, repeated: true },
  { no: 10, name: "reserved_name", kind: "scalar", T: 9, repeated: true }
]);
var DescriptorProto_ExtensionRange = class _DescriptorProto_ExtensionRange extends Message {
  constructor(data) {
    super();
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _DescriptorProto_ExtensionRange().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _DescriptorProto_ExtensionRange().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _DescriptorProto_ExtensionRange().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_DescriptorProto_ExtensionRange, a, b);
  }
};
DescriptorProto_ExtensionRange.runtime = proto2;
DescriptorProto_ExtensionRange.typeName = "google.protobuf.DescriptorProto.ExtensionRange";
DescriptorProto_ExtensionRange.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "start", kind: "scalar", T: 5, opt: true },
  { no: 2, name: "end", kind: "scalar", T: 5, opt: true },
  { no: 3, name: "options", kind: "message", T: ExtensionRangeOptions, opt: true }
]);
var DescriptorProto_ReservedRange = class _DescriptorProto_ReservedRange extends Message {
  constructor(data) {
    super();
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _DescriptorProto_ReservedRange().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _DescriptorProto_ReservedRange().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _DescriptorProto_ReservedRange().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_DescriptorProto_ReservedRange, a, b);
  }
};
DescriptorProto_ReservedRange.runtime = proto2;
DescriptorProto_ReservedRange.typeName = "google.protobuf.DescriptorProto.ReservedRange";
DescriptorProto_ReservedRange.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "start", kind: "scalar", T: 5, opt: true },
  { no: 2, name: "end", kind: "scalar", T: 5, opt: true }
]);
var ExtensionRangeOptions = class _ExtensionRangeOptions extends Message {
  constructor(data) {
    super();
    this.uninterpretedOption = [];
    this.declaration = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _ExtensionRangeOptions().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _ExtensionRangeOptions().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _ExtensionRangeOptions().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_ExtensionRangeOptions, a, b);
  }
};
ExtensionRangeOptions.runtime = proto2;
ExtensionRangeOptions.typeName = "google.protobuf.ExtensionRangeOptions";
ExtensionRangeOptions.fields = proto2.util.newFieldList(() => [
  { no: 999, name: "uninterpreted_option", kind: "message", T: UninterpretedOption, repeated: true },
  { no: 2, name: "declaration", kind: "message", T: ExtensionRangeOptions_Declaration, repeated: true },
  { no: 3, name: "verification", kind: "enum", T: proto2.getEnumType(ExtensionRangeOptions_VerificationState), opt: true, default: ExtensionRangeOptions_VerificationState.UNVERIFIED }
]);
var ExtensionRangeOptions_VerificationState;
(function(ExtensionRangeOptions_VerificationState2) {
  ExtensionRangeOptions_VerificationState2[ExtensionRangeOptions_VerificationState2["DECLARATION"] = 0] = "DECLARATION";
  ExtensionRangeOptions_VerificationState2[ExtensionRangeOptions_VerificationState2["UNVERIFIED"] = 1] = "UNVERIFIED";
})(ExtensionRangeOptions_VerificationState || (ExtensionRangeOptions_VerificationState = {}));
proto2.util.setEnumType(ExtensionRangeOptions_VerificationState, "google.protobuf.ExtensionRangeOptions.VerificationState", [
  { no: 0, name: "DECLARATION" },
  { no: 1, name: "UNVERIFIED" }
]);
var ExtensionRangeOptions_Declaration = class _ExtensionRangeOptions_Declaration extends Message {
  constructor(data) {
    super();
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _ExtensionRangeOptions_Declaration().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _ExtensionRangeOptions_Declaration().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _ExtensionRangeOptions_Declaration().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_ExtensionRangeOptions_Declaration, a, b);
  }
};
ExtensionRangeOptions_Declaration.runtime = proto2;
ExtensionRangeOptions_Declaration.typeName = "google.protobuf.ExtensionRangeOptions.Declaration";
ExtensionRangeOptions_Declaration.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "number", kind: "scalar", T: 5, opt: true },
  { no: 2, name: "full_name", kind: "scalar", T: 9, opt: true },
  { no: 3, name: "type", kind: "scalar", T: 9, opt: true },
  { no: 4, name: "is_repeated", kind: "scalar", T: 8, opt: true },
  { no: 5, name: "reserved", kind: "scalar", T: 8, opt: true },
  { no: 6, name: "repeated", kind: "scalar", T: 8, opt: true }
]);
var FieldDescriptorProto = class _FieldDescriptorProto extends Message {
  constructor(data) {
    super();
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _FieldDescriptorProto().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _FieldDescriptorProto().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _FieldDescriptorProto().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_FieldDescriptorProto, a, b);
  }
};
FieldDescriptorProto.runtime = proto2;
FieldDescriptorProto.typeName = "google.protobuf.FieldDescriptorProto";
FieldDescriptorProto.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "name", kind: "scalar", T: 9, opt: true },
  { no: 3, name: "number", kind: "scalar", T: 5, opt: true },
  { no: 4, name: "label", kind: "enum", T: proto2.getEnumType(FieldDescriptorProto_Label), opt: true },
  { no: 5, name: "type", kind: "enum", T: proto2.getEnumType(FieldDescriptorProto_Type), opt: true },
  { no: 6, name: "type_name", kind: "scalar", T: 9, opt: true },
  { no: 2, name: "extendee", kind: "scalar", T: 9, opt: true },
  { no: 7, name: "default_value", kind: "scalar", T: 9, opt: true },
  { no: 9, name: "oneof_index", kind: "scalar", T: 5, opt: true },
  { no: 10, name: "json_name", kind: "scalar", T: 9, opt: true },
  { no: 8, name: "options", kind: "message", T: FieldOptions, opt: true },
  { no: 17, name: "proto3_optional", kind: "scalar", T: 8, opt: true }
]);
var FieldDescriptorProto_Type;
(function(FieldDescriptorProto_Type2) {
  FieldDescriptorProto_Type2[FieldDescriptorProto_Type2["DOUBLE"] = 1] = "DOUBLE";
  FieldDescriptorProto_Type2[FieldDescriptorProto_Type2["FLOAT"] = 2] = "FLOAT";
  FieldDescriptorProto_Type2[FieldDescriptorProto_Type2["INT64"] = 3] = "INT64";
  FieldDescriptorProto_Type2[FieldDescriptorProto_Type2["UINT64"] = 4] = "UINT64";
  FieldDescriptorProto_Type2[FieldDescriptorProto_Type2["INT32"] = 5] = "INT32";
  FieldDescriptorProto_Type2[FieldDescriptorProto_Type2["FIXED64"] = 6] = "FIXED64";
  FieldDescriptorProto_Type2[FieldDescriptorProto_Type2["FIXED32"] = 7] = "FIXED32";
  FieldDescriptorProto_Type2[FieldDescriptorProto_Type2["BOOL"] = 8] = "BOOL";
  FieldDescriptorProto_Type2[FieldDescriptorProto_Type2["STRING"] = 9] = "STRING";
  FieldDescriptorProto_Type2[FieldDescriptorProto_Type2["GROUP"] = 10] = "GROUP";
  FieldDescriptorProto_Type2[FieldDescriptorProto_Type2["MESSAGE"] = 11] = "MESSAGE";
  FieldDescriptorProto_Type2[FieldDescriptorProto_Type2["BYTES"] = 12] = "BYTES";
  FieldDescriptorProto_Type2[FieldDescriptorProto_Type2["UINT32"] = 13] = "UINT32";
  FieldDescriptorProto_Type2[FieldDescriptorProto_Type2["ENUM"] = 14] = "ENUM";
  FieldDescriptorProto_Type2[FieldDescriptorProto_Type2["SFIXED32"] = 15] = "SFIXED32";
  FieldDescriptorProto_Type2[FieldDescriptorProto_Type2["SFIXED64"] = 16] = "SFIXED64";
  FieldDescriptorProto_Type2[FieldDescriptorProto_Type2["SINT32"] = 17] = "SINT32";
  FieldDescriptorProto_Type2[FieldDescriptorProto_Type2["SINT64"] = 18] = "SINT64";
})(FieldDescriptorProto_Type || (FieldDescriptorProto_Type = {}));
proto2.util.setEnumType(FieldDescriptorProto_Type, "google.protobuf.FieldDescriptorProto.Type", [
  { no: 1, name: "TYPE_DOUBLE" },
  { no: 2, name: "TYPE_FLOAT" },
  { no: 3, name: "TYPE_INT64" },
  { no: 4, name: "TYPE_UINT64" },
  { no: 5, name: "TYPE_INT32" },
  { no: 6, name: "TYPE_FIXED64" },
  { no: 7, name: "TYPE_FIXED32" },
  { no: 8, name: "TYPE_BOOL" },
  { no: 9, name: "TYPE_STRING" },
  { no: 10, name: "TYPE_GROUP" },
  { no: 11, name: "TYPE_MESSAGE" },
  { no: 12, name: "TYPE_BYTES" },
  { no: 13, name: "TYPE_UINT32" },
  { no: 14, name: "TYPE_ENUM" },
  { no: 15, name: "TYPE_SFIXED32" },
  { no: 16, name: "TYPE_SFIXED64" },
  { no: 17, name: "TYPE_SINT32" },
  { no: 18, name: "TYPE_SINT64" }
]);
var FieldDescriptorProto_Label;
(function(FieldDescriptorProto_Label2) {
  FieldDescriptorProto_Label2[FieldDescriptorProto_Label2["OPTIONAL"] = 1] = "OPTIONAL";
  FieldDescriptorProto_Label2[FieldDescriptorProto_Label2["REQUIRED"] = 2] = "REQUIRED";
  FieldDescriptorProto_Label2[FieldDescriptorProto_Label2["REPEATED"] = 3] = "REPEATED";
})(FieldDescriptorProto_Label || (FieldDescriptorProto_Label = {}));
proto2.util.setEnumType(FieldDescriptorProto_Label, "google.protobuf.FieldDescriptorProto.Label", [
  { no: 1, name: "LABEL_OPTIONAL" },
  { no: 2, name: "LABEL_REQUIRED" },
  { no: 3, name: "LABEL_REPEATED" }
]);
var OneofDescriptorProto = class _OneofDescriptorProto extends Message {
  constructor(data) {
    super();
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _OneofDescriptorProto().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _OneofDescriptorProto().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _OneofDescriptorProto().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_OneofDescriptorProto, a, b);
  }
};
OneofDescriptorProto.runtime = proto2;
OneofDescriptorProto.typeName = "google.protobuf.OneofDescriptorProto";
OneofDescriptorProto.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "name", kind: "scalar", T: 9, opt: true },
  { no: 2, name: "options", kind: "message", T: OneofOptions, opt: true }
]);
var EnumDescriptorProto = class _EnumDescriptorProto extends Message {
  constructor(data) {
    super();
    this.value = [];
    this.reservedRange = [];
    this.reservedName = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _EnumDescriptorProto().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _EnumDescriptorProto().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _EnumDescriptorProto().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_EnumDescriptorProto, a, b);
  }
};
EnumDescriptorProto.runtime = proto2;
EnumDescriptorProto.typeName = "google.protobuf.EnumDescriptorProto";
EnumDescriptorProto.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "name", kind: "scalar", T: 9, opt: true },
  { no: 2, name: "value", kind: "message", T: EnumValueDescriptorProto, repeated: true },
  { no: 3, name: "options", kind: "message", T: EnumOptions, opt: true },
  { no: 4, name: "reserved_range", kind: "message", T: EnumDescriptorProto_EnumReservedRange, repeated: true },
  { no: 5, name: "reserved_name", kind: "scalar", T: 9, repeated: true }
]);
var EnumDescriptorProto_EnumReservedRange = class _EnumDescriptorProto_EnumReservedRange extends Message {
  constructor(data) {
    super();
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _EnumDescriptorProto_EnumReservedRange().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _EnumDescriptorProto_EnumReservedRange().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _EnumDescriptorProto_EnumReservedRange().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_EnumDescriptorProto_EnumReservedRange, a, b);
  }
};
EnumDescriptorProto_EnumReservedRange.runtime = proto2;
EnumDescriptorProto_EnumReservedRange.typeName = "google.protobuf.EnumDescriptorProto.EnumReservedRange";
EnumDescriptorProto_EnumReservedRange.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "start", kind: "scalar", T: 5, opt: true },
  { no: 2, name: "end", kind: "scalar", T: 5, opt: true }
]);
var EnumValueDescriptorProto = class _EnumValueDescriptorProto extends Message {
  constructor(data) {
    super();
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _EnumValueDescriptorProto().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _EnumValueDescriptorProto().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _EnumValueDescriptorProto().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_EnumValueDescriptorProto, a, b);
  }
};
EnumValueDescriptorProto.runtime = proto2;
EnumValueDescriptorProto.typeName = "google.protobuf.EnumValueDescriptorProto";
EnumValueDescriptorProto.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "name", kind: "scalar", T: 9, opt: true },
  { no: 2, name: "number", kind: "scalar", T: 5, opt: true },
  { no: 3, name: "options", kind: "message", T: EnumValueOptions, opt: true }
]);
var ServiceDescriptorProto = class _ServiceDescriptorProto extends Message {
  constructor(data) {
    super();
    this.method = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _ServiceDescriptorProto().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _ServiceDescriptorProto().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _ServiceDescriptorProto().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_ServiceDescriptorProto, a, b);
  }
};
ServiceDescriptorProto.runtime = proto2;
ServiceDescriptorProto.typeName = "google.protobuf.ServiceDescriptorProto";
ServiceDescriptorProto.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "name", kind: "scalar", T: 9, opt: true },
  { no: 2, name: "method", kind: "message", T: MethodDescriptorProto, repeated: true },
  { no: 3, name: "options", kind: "message", T: ServiceOptions, opt: true }
]);
var MethodDescriptorProto = class _MethodDescriptorProto extends Message {
  constructor(data) {
    super();
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _MethodDescriptorProto().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _MethodDescriptorProto().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _MethodDescriptorProto().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_MethodDescriptorProto, a, b);
  }
};
MethodDescriptorProto.runtime = proto2;
MethodDescriptorProto.typeName = "google.protobuf.MethodDescriptorProto";
MethodDescriptorProto.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "name", kind: "scalar", T: 9, opt: true },
  { no: 2, name: "input_type", kind: "scalar", T: 9, opt: true },
  { no: 3, name: "output_type", kind: "scalar", T: 9, opt: true },
  { no: 4, name: "options", kind: "message", T: MethodOptions, opt: true },
  { no: 5, name: "client_streaming", kind: "scalar", T: 8, opt: true, default: false },
  { no: 6, name: "server_streaming", kind: "scalar", T: 8, opt: true, default: false }
]);
var FileOptions = class _FileOptions extends Message {
  constructor(data) {
    super();
    this.uninterpretedOption = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _FileOptions().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _FileOptions().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _FileOptions().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_FileOptions, a, b);
  }
};
FileOptions.runtime = proto2;
FileOptions.typeName = "google.protobuf.FileOptions";
FileOptions.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "java_package", kind: "scalar", T: 9, opt: true },
  { no: 8, name: "java_outer_classname", kind: "scalar", T: 9, opt: true },
  { no: 10, name: "java_multiple_files", kind: "scalar", T: 8, opt: true, default: false },
  { no: 20, name: "java_generate_equals_and_hash", kind: "scalar", T: 8, opt: true },
  { no: 27, name: "java_string_check_utf8", kind: "scalar", T: 8, opt: true, default: false },
  { no: 9, name: "optimize_for", kind: "enum", T: proto2.getEnumType(FileOptions_OptimizeMode), opt: true, default: FileOptions_OptimizeMode.SPEED },
  { no: 11, name: "go_package", kind: "scalar", T: 9, opt: true },
  { no: 16, name: "cc_generic_services", kind: "scalar", T: 8, opt: true, default: false },
  { no: 17, name: "java_generic_services", kind: "scalar", T: 8, opt: true, default: false },
  { no: 18, name: "py_generic_services", kind: "scalar", T: 8, opt: true, default: false },
  { no: 42, name: "php_generic_services", kind: "scalar", T: 8, opt: true, default: false },
  { no: 23, name: "deprecated", kind: "scalar", T: 8, opt: true, default: false },
  { no: 31, name: "cc_enable_arenas", kind: "scalar", T: 8, opt: true, default: true },
  { no: 36, name: "objc_class_prefix", kind: "scalar", T: 9, opt: true },
  { no: 37, name: "csharp_namespace", kind: "scalar", T: 9, opt: true },
  { no: 39, name: "swift_prefix", kind: "scalar", T: 9, opt: true },
  { no: 40, name: "php_class_prefix", kind: "scalar", T: 9, opt: true },
  { no: 41, name: "php_namespace", kind: "scalar", T: 9, opt: true },
  { no: 44, name: "php_metadata_namespace", kind: "scalar", T: 9, opt: true },
  { no: 45, name: "ruby_package", kind: "scalar", T: 9, opt: true },
  { no: 999, name: "uninterpreted_option", kind: "message", T: UninterpretedOption, repeated: true }
]);
var FileOptions_OptimizeMode;
(function(FileOptions_OptimizeMode2) {
  FileOptions_OptimizeMode2[FileOptions_OptimizeMode2["SPEED"] = 1] = "SPEED";
  FileOptions_OptimizeMode2[FileOptions_OptimizeMode2["CODE_SIZE"] = 2] = "CODE_SIZE";
  FileOptions_OptimizeMode2[FileOptions_OptimizeMode2["LITE_RUNTIME"] = 3] = "LITE_RUNTIME";
})(FileOptions_OptimizeMode || (FileOptions_OptimizeMode = {}));
proto2.util.setEnumType(FileOptions_OptimizeMode, "google.protobuf.FileOptions.OptimizeMode", [
  { no: 1, name: "SPEED" },
  { no: 2, name: "CODE_SIZE" },
  { no: 3, name: "LITE_RUNTIME" }
]);
var MessageOptions = class _MessageOptions extends Message {
  constructor(data) {
    super();
    this.uninterpretedOption = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _MessageOptions().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _MessageOptions().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _MessageOptions().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_MessageOptions, a, b);
  }
};
MessageOptions.runtime = proto2;
MessageOptions.typeName = "google.protobuf.MessageOptions";
MessageOptions.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "message_set_wire_format", kind: "scalar", T: 8, opt: true, default: false },
  { no: 2, name: "no_standard_descriptor_accessor", kind: "scalar", T: 8, opt: true, default: false },
  { no: 3, name: "deprecated", kind: "scalar", T: 8, opt: true, default: false },
  { no: 7, name: "map_entry", kind: "scalar", T: 8, opt: true },
  { no: 11, name: "deprecated_legacy_json_field_conflicts", kind: "scalar", T: 8, opt: true },
  { no: 999, name: "uninterpreted_option", kind: "message", T: UninterpretedOption, repeated: true }
]);
var FieldOptions = class _FieldOptions extends Message {
  constructor(data) {
    super();
    this.targets = [];
    this.uninterpretedOption = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _FieldOptions().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _FieldOptions().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _FieldOptions().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_FieldOptions, a, b);
  }
};
FieldOptions.runtime = proto2;
FieldOptions.typeName = "google.protobuf.FieldOptions";
FieldOptions.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "ctype", kind: "enum", T: proto2.getEnumType(FieldOptions_CType), opt: true, default: FieldOptions_CType.STRING },
  { no: 2, name: "packed", kind: "scalar", T: 8, opt: true },
  { no: 6, name: "jstype", kind: "enum", T: proto2.getEnumType(FieldOptions_JSType), opt: true, default: FieldOptions_JSType.JS_NORMAL },
  { no: 5, name: "lazy", kind: "scalar", T: 8, opt: true, default: false },
  { no: 15, name: "unverified_lazy", kind: "scalar", T: 8, opt: true, default: false },
  { no: 3, name: "deprecated", kind: "scalar", T: 8, opt: true, default: false },
  { no: 10, name: "weak", kind: "scalar", T: 8, opt: true, default: false },
  { no: 16, name: "debug_redact", kind: "scalar", T: 8, opt: true, default: false },
  { no: 17, name: "retention", kind: "enum", T: proto2.getEnumType(FieldOptions_OptionRetention), opt: true },
  { no: 18, name: "target", kind: "enum", T: proto2.getEnumType(FieldOptions_OptionTargetType), opt: true },
  { no: 19, name: "targets", kind: "enum", T: proto2.getEnumType(FieldOptions_OptionTargetType), repeated: true },
  { no: 999, name: "uninterpreted_option", kind: "message", T: UninterpretedOption, repeated: true }
]);
var FieldOptions_CType;
(function(FieldOptions_CType2) {
  FieldOptions_CType2[FieldOptions_CType2["STRING"] = 0] = "STRING";
  FieldOptions_CType2[FieldOptions_CType2["CORD"] = 1] = "CORD";
  FieldOptions_CType2[FieldOptions_CType2["STRING_PIECE"] = 2] = "STRING_PIECE";
})(FieldOptions_CType || (FieldOptions_CType = {}));
proto2.util.setEnumType(FieldOptions_CType, "google.protobuf.FieldOptions.CType", [
  { no: 0, name: "STRING" },
  { no: 1, name: "CORD" },
  { no: 2, name: "STRING_PIECE" }
]);
var FieldOptions_JSType;
(function(FieldOptions_JSType2) {
  FieldOptions_JSType2[FieldOptions_JSType2["JS_NORMAL"] = 0] = "JS_NORMAL";
  FieldOptions_JSType2[FieldOptions_JSType2["JS_STRING"] = 1] = "JS_STRING";
  FieldOptions_JSType2[FieldOptions_JSType2["JS_NUMBER"] = 2] = "JS_NUMBER";
})(FieldOptions_JSType || (FieldOptions_JSType = {}));
proto2.util.setEnumType(FieldOptions_JSType, "google.protobuf.FieldOptions.JSType", [
  { no: 0, name: "JS_NORMAL" },
  { no: 1, name: "JS_STRING" },
  { no: 2, name: "JS_NUMBER" }
]);
var FieldOptions_OptionRetention;
(function(FieldOptions_OptionRetention2) {
  FieldOptions_OptionRetention2[FieldOptions_OptionRetention2["RETENTION_UNKNOWN"] = 0] = "RETENTION_UNKNOWN";
  FieldOptions_OptionRetention2[FieldOptions_OptionRetention2["RETENTION_RUNTIME"] = 1] = "RETENTION_RUNTIME";
  FieldOptions_OptionRetention2[FieldOptions_OptionRetention2["RETENTION_SOURCE"] = 2] = "RETENTION_SOURCE";
})(FieldOptions_OptionRetention || (FieldOptions_OptionRetention = {}));
proto2.util.setEnumType(FieldOptions_OptionRetention, "google.protobuf.FieldOptions.OptionRetention", [
  { no: 0, name: "RETENTION_UNKNOWN" },
  { no: 1, name: "RETENTION_RUNTIME" },
  { no: 2, name: "RETENTION_SOURCE" }
]);
var FieldOptions_OptionTargetType;
(function(FieldOptions_OptionTargetType2) {
  FieldOptions_OptionTargetType2[FieldOptions_OptionTargetType2["TARGET_TYPE_UNKNOWN"] = 0] = "TARGET_TYPE_UNKNOWN";
  FieldOptions_OptionTargetType2[FieldOptions_OptionTargetType2["TARGET_TYPE_FILE"] = 1] = "TARGET_TYPE_FILE";
  FieldOptions_OptionTargetType2[FieldOptions_OptionTargetType2["TARGET_TYPE_EXTENSION_RANGE"] = 2] = "TARGET_TYPE_EXTENSION_RANGE";
  FieldOptions_OptionTargetType2[FieldOptions_OptionTargetType2["TARGET_TYPE_MESSAGE"] = 3] = "TARGET_TYPE_MESSAGE";
  FieldOptions_OptionTargetType2[FieldOptions_OptionTargetType2["TARGET_TYPE_FIELD"] = 4] = "TARGET_TYPE_FIELD";
  FieldOptions_OptionTargetType2[FieldOptions_OptionTargetType2["TARGET_TYPE_ONEOF"] = 5] = "TARGET_TYPE_ONEOF";
  FieldOptions_OptionTargetType2[FieldOptions_OptionTargetType2["TARGET_TYPE_ENUM"] = 6] = "TARGET_TYPE_ENUM";
  FieldOptions_OptionTargetType2[FieldOptions_OptionTargetType2["TARGET_TYPE_ENUM_ENTRY"] = 7] = "TARGET_TYPE_ENUM_ENTRY";
  FieldOptions_OptionTargetType2[FieldOptions_OptionTargetType2["TARGET_TYPE_SERVICE"] = 8] = "TARGET_TYPE_SERVICE";
  FieldOptions_OptionTargetType2[FieldOptions_OptionTargetType2["TARGET_TYPE_METHOD"] = 9] = "TARGET_TYPE_METHOD";
})(FieldOptions_OptionTargetType || (FieldOptions_OptionTargetType = {}));
proto2.util.setEnumType(FieldOptions_OptionTargetType, "google.protobuf.FieldOptions.OptionTargetType", [
  { no: 0, name: "TARGET_TYPE_UNKNOWN" },
  { no: 1, name: "TARGET_TYPE_FILE" },
  { no: 2, name: "TARGET_TYPE_EXTENSION_RANGE" },
  { no: 3, name: "TARGET_TYPE_MESSAGE" },
  { no: 4, name: "TARGET_TYPE_FIELD" },
  { no: 5, name: "TARGET_TYPE_ONEOF" },
  { no: 6, name: "TARGET_TYPE_ENUM" },
  { no: 7, name: "TARGET_TYPE_ENUM_ENTRY" },
  { no: 8, name: "TARGET_TYPE_SERVICE" },
  { no: 9, name: "TARGET_TYPE_METHOD" }
]);
var OneofOptions = class _OneofOptions extends Message {
  constructor(data) {
    super();
    this.uninterpretedOption = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _OneofOptions().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _OneofOptions().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _OneofOptions().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_OneofOptions, a, b);
  }
};
OneofOptions.runtime = proto2;
OneofOptions.typeName = "google.protobuf.OneofOptions";
OneofOptions.fields = proto2.util.newFieldList(() => [
  { no: 999, name: "uninterpreted_option", kind: "message", T: UninterpretedOption, repeated: true }
]);
var EnumOptions = class _EnumOptions extends Message {
  constructor(data) {
    super();
    this.uninterpretedOption = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _EnumOptions().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _EnumOptions().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _EnumOptions().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_EnumOptions, a, b);
  }
};
EnumOptions.runtime = proto2;
EnumOptions.typeName = "google.protobuf.EnumOptions";
EnumOptions.fields = proto2.util.newFieldList(() => [
  { no: 2, name: "allow_alias", kind: "scalar", T: 8, opt: true },
  { no: 3, name: "deprecated", kind: "scalar", T: 8, opt: true, default: false },
  { no: 6, name: "deprecated_legacy_json_field_conflicts", kind: "scalar", T: 8, opt: true },
  { no: 999, name: "uninterpreted_option", kind: "message", T: UninterpretedOption, repeated: true }
]);
var EnumValueOptions = class _EnumValueOptions extends Message {
  constructor(data) {
    super();
    this.uninterpretedOption = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _EnumValueOptions().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _EnumValueOptions().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _EnumValueOptions().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_EnumValueOptions, a, b);
  }
};
EnumValueOptions.runtime = proto2;
EnumValueOptions.typeName = "google.protobuf.EnumValueOptions";
EnumValueOptions.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "deprecated", kind: "scalar", T: 8, opt: true, default: false },
  { no: 999, name: "uninterpreted_option", kind: "message", T: UninterpretedOption, repeated: true }
]);
var ServiceOptions = class _ServiceOptions extends Message {
  constructor(data) {
    super();
    this.uninterpretedOption = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _ServiceOptions().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _ServiceOptions().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _ServiceOptions().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_ServiceOptions, a, b);
  }
};
ServiceOptions.runtime = proto2;
ServiceOptions.typeName = "google.protobuf.ServiceOptions";
ServiceOptions.fields = proto2.util.newFieldList(() => [
  { no: 33, name: "deprecated", kind: "scalar", T: 8, opt: true, default: false },
  { no: 999, name: "uninterpreted_option", kind: "message", T: UninterpretedOption, repeated: true }
]);
var MethodOptions = class _MethodOptions extends Message {
  constructor(data) {
    super();
    this.uninterpretedOption = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _MethodOptions().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _MethodOptions().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _MethodOptions().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_MethodOptions, a, b);
  }
};
MethodOptions.runtime = proto2;
MethodOptions.typeName = "google.protobuf.MethodOptions";
MethodOptions.fields = proto2.util.newFieldList(() => [
  { no: 33, name: "deprecated", kind: "scalar", T: 8, opt: true, default: false },
  { no: 34, name: "idempotency_level", kind: "enum", T: proto2.getEnumType(MethodOptions_IdempotencyLevel), opt: true, default: MethodOptions_IdempotencyLevel.IDEMPOTENCY_UNKNOWN },
  { no: 999, name: "uninterpreted_option", kind: "message", T: UninterpretedOption, repeated: true }
]);
var MethodOptions_IdempotencyLevel;
(function(MethodOptions_IdempotencyLevel2) {
  MethodOptions_IdempotencyLevel2[MethodOptions_IdempotencyLevel2["IDEMPOTENCY_UNKNOWN"] = 0] = "IDEMPOTENCY_UNKNOWN";
  MethodOptions_IdempotencyLevel2[MethodOptions_IdempotencyLevel2["NO_SIDE_EFFECTS"] = 1] = "NO_SIDE_EFFECTS";
  MethodOptions_IdempotencyLevel2[MethodOptions_IdempotencyLevel2["IDEMPOTENT"] = 2] = "IDEMPOTENT";
})(MethodOptions_IdempotencyLevel || (MethodOptions_IdempotencyLevel = {}));
proto2.util.setEnumType(MethodOptions_IdempotencyLevel, "google.protobuf.MethodOptions.IdempotencyLevel", [
  { no: 0, name: "IDEMPOTENCY_UNKNOWN" },
  { no: 1, name: "NO_SIDE_EFFECTS" },
  { no: 2, name: "IDEMPOTENT" }
]);
var UninterpretedOption = class _UninterpretedOption extends Message {
  constructor(data) {
    super();
    this.name = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _UninterpretedOption().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _UninterpretedOption().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _UninterpretedOption().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_UninterpretedOption, a, b);
  }
};
UninterpretedOption.runtime = proto2;
UninterpretedOption.typeName = "google.protobuf.UninterpretedOption";
UninterpretedOption.fields = proto2.util.newFieldList(() => [
  { no: 2, name: "name", kind: "message", T: UninterpretedOption_NamePart, repeated: true },
  { no: 3, name: "identifier_value", kind: "scalar", T: 9, opt: true },
  { no: 4, name: "positive_int_value", kind: "scalar", T: 4, opt: true },
  { no: 5, name: "negative_int_value", kind: "scalar", T: 3, opt: true },
  { no: 6, name: "double_value", kind: "scalar", T: 1, opt: true },
  { no: 7, name: "string_value", kind: "scalar", T: 12, opt: true },
  { no: 8, name: "aggregate_value", kind: "scalar", T: 9, opt: true }
]);
var UninterpretedOption_NamePart = class _UninterpretedOption_NamePart extends Message {
  constructor(data) {
    super();
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _UninterpretedOption_NamePart().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _UninterpretedOption_NamePart().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _UninterpretedOption_NamePart().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_UninterpretedOption_NamePart, a, b);
  }
};
UninterpretedOption_NamePart.runtime = proto2;
UninterpretedOption_NamePart.typeName = "google.protobuf.UninterpretedOption.NamePart";
UninterpretedOption_NamePart.fields = proto2.util.newFieldList(() => [
  {
    no: 1,
    name: "name_part",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  },
  {
    no: 2,
    name: "is_extension",
    kind: "scalar",
    T: 8
    /* ScalarType.BOOL */
  }
]);
var SourceCodeInfo = class _SourceCodeInfo extends Message {
  constructor(data) {
    super();
    this.location = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _SourceCodeInfo().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _SourceCodeInfo().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _SourceCodeInfo().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_SourceCodeInfo, a, b);
  }
};
SourceCodeInfo.runtime = proto2;
SourceCodeInfo.typeName = "google.protobuf.SourceCodeInfo";
SourceCodeInfo.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "location", kind: "message", T: SourceCodeInfo_Location, repeated: true }
]);
var SourceCodeInfo_Location = class _SourceCodeInfo_Location extends Message {
  constructor(data) {
    super();
    this.path = [];
    this.span = [];
    this.leadingDetachedComments = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _SourceCodeInfo_Location().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _SourceCodeInfo_Location().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _SourceCodeInfo_Location().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_SourceCodeInfo_Location, a, b);
  }
};
SourceCodeInfo_Location.runtime = proto2;
SourceCodeInfo_Location.typeName = "google.protobuf.SourceCodeInfo.Location";
SourceCodeInfo_Location.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "path", kind: "scalar", T: 5, repeated: true, packed: true },
  { no: 2, name: "span", kind: "scalar", T: 5, repeated: true, packed: true },
  { no: 3, name: "leading_comments", kind: "scalar", T: 9, opt: true },
  { no: 4, name: "trailing_comments", kind: "scalar", T: 9, opt: true },
  { no: 6, name: "leading_detached_comments", kind: "scalar", T: 9, repeated: true }
]);
var GeneratedCodeInfo = class _GeneratedCodeInfo extends Message {
  constructor(data) {
    super();
    this.annotation = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _GeneratedCodeInfo().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _GeneratedCodeInfo().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _GeneratedCodeInfo().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_GeneratedCodeInfo, a, b);
  }
};
GeneratedCodeInfo.runtime = proto2;
GeneratedCodeInfo.typeName = "google.protobuf.GeneratedCodeInfo";
GeneratedCodeInfo.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "annotation", kind: "message", T: GeneratedCodeInfo_Annotation, repeated: true }
]);
var GeneratedCodeInfo_Annotation = class _GeneratedCodeInfo_Annotation extends Message {
  constructor(data) {
    super();
    this.path = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _GeneratedCodeInfo_Annotation().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _GeneratedCodeInfo_Annotation().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _GeneratedCodeInfo_Annotation().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_GeneratedCodeInfo_Annotation, a, b);
  }
};
GeneratedCodeInfo_Annotation.runtime = proto2;
GeneratedCodeInfo_Annotation.typeName = "google.protobuf.GeneratedCodeInfo.Annotation";
GeneratedCodeInfo_Annotation.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "path", kind: "scalar", T: 5, repeated: true, packed: true },
  { no: 2, name: "source_file", kind: "scalar", T: 9, opt: true },
  { no: 3, name: "begin", kind: "scalar", T: 5, opt: true },
  { no: 4, name: "end", kind: "scalar", T: 5, opt: true },
  { no: 5, name: "semantic", kind: "enum", T: proto2.getEnumType(GeneratedCodeInfo_Annotation_Semantic), opt: true }
]);
var GeneratedCodeInfo_Annotation_Semantic;
(function(GeneratedCodeInfo_Annotation_Semantic2) {
  GeneratedCodeInfo_Annotation_Semantic2[GeneratedCodeInfo_Annotation_Semantic2["NONE"] = 0] = "NONE";
  GeneratedCodeInfo_Annotation_Semantic2[GeneratedCodeInfo_Annotation_Semantic2["SET"] = 1] = "SET";
  GeneratedCodeInfo_Annotation_Semantic2[GeneratedCodeInfo_Annotation_Semantic2["ALIAS"] = 2] = "ALIAS";
})(GeneratedCodeInfo_Annotation_Semantic || (GeneratedCodeInfo_Annotation_Semantic = {}));
proto2.util.setEnumType(GeneratedCodeInfo_Annotation_Semantic, "google.protobuf.GeneratedCodeInfo.Annotation.Semantic", [
  { no: 0, name: "NONE" },
  { no: 1, name: "SET" },
  { no: 2, name: "ALIAS" }
]);

// node_modules/@bufbuild/protobuf/dist/esm/create-descriptor-set.js
function createDescriptorSet(input) {
  const cart = {
    enums: /* @__PURE__ */ new Map(),
    messages: /* @__PURE__ */ new Map(),
    services: /* @__PURE__ */ new Map(),
    extensions: /* @__PURE__ */ new Map(),
    mapEntries: /* @__PURE__ */ new Map()
  };
  const fileDescriptors = input instanceof FileDescriptorSet ? input.file : input instanceof Uint8Array ? FileDescriptorSet.fromBinary(input).file : input;
  const files = fileDescriptors.map((proto) => newFile(proto, cart));
  return Object.assign({ files }, cart);
}
function newFile(proto, cart) {
  var _a, _b, _c;
  assert(proto.name, `invalid FileDescriptorProto: missing name`);
  assert(proto.syntax === void 0 || proto.syntax === "proto3", `invalid FileDescriptorProto: unsupported syntax: ${(_a = proto.syntax) !== null && _a !== void 0 ? _a : "undefined"}`);
  const file = {
    kind: "file",
    proto,
    deprecated: (_c = (_b = proto.options) === null || _b === void 0 ? void 0 : _b.deprecated) !== null && _c !== void 0 ? _c : false,
    syntax: proto.syntax === "proto3" ? "proto3" : "proto2",
    name: proto.name.replace(/\.proto/, ""),
    enums: [],
    messages: [],
    extensions: [],
    services: [],
    toString() {
      return `file ${this.proto.name}`;
    },
    getSyntaxComments() {
      return findComments(this.proto.sourceCodeInfo, [
        FieldNumber.FileDescriptorProto_Syntax
      ]);
    },
    getPackageComments() {
      return findComments(this.proto.sourceCodeInfo, [
        FieldNumber.FileDescriptorProto_Package
      ]);
    }
  };
  cart.mapEntries.clear();
  for (const enumProto of proto.enumType) {
    addEnum(enumProto, file, void 0, cart);
  }
  for (const messageProto of proto.messageType) {
    addMessage(messageProto, file, void 0, cart);
  }
  for (const serviceProto of proto.service) {
    addService(serviceProto, file, cart);
  }
  addExtensions(file, cart);
  for (const mapEntry of cart.mapEntries.values()) {
    addFields(mapEntry, cart);
  }
  for (const message of file.messages) {
    addFields(message, cart);
    addExtensions(message, cart);
  }
  cart.mapEntries.clear();
  return file;
}
function addExtensions(desc, cart) {
  switch (desc.kind) {
    case "file":
      for (const proto of desc.proto.extension) {
        const ext = newExtension(proto, desc, void 0, cart);
        desc.extensions.push(ext);
        cart.extensions.set(ext.typeName, ext);
      }
      break;
    case "message":
      for (const proto of desc.proto.extension) {
        const ext = newExtension(proto, desc.file, desc, cart);
        desc.nestedExtensions.push(ext);
        cart.extensions.set(ext.typeName, ext);
      }
      for (const message of desc.nestedMessages) {
        addExtensions(message, cart);
      }
      break;
  }
}
function addFields(message, cart) {
  const allOneofs = message.proto.oneofDecl.map((proto) => newOneof(proto, message));
  const oneofsSeen = /* @__PURE__ */ new Set();
  for (const proto of message.proto.field) {
    const oneof = findOneof(proto, allOneofs);
    const field = newField(proto, message.file, message, oneof, cart);
    message.fields.push(field);
    if (oneof === void 0) {
      message.members.push(field);
    } else {
      oneof.fields.push(field);
      if (!oneofsSeen.has(oneof)) {
        oneofsSeen.add(oneof);
        message.members.push(oneof);
      }
    }
  }
  for (const oneof of allOneofs.filter((o) => oneofsSeen.has(o))) {
    message.oneofs.push(oneof);
  }
  for (const child of message.nestedMessages) {
    addFields(child, cart);
  }
}
function addEnum(proto, file, parent, cart) {
  var _a, _b, _c;
  assert(proto.name, `invalid EnumDescriptorProto: missing name`);
  const desc = {
    kind: "enum",
    proto,
    deprecated: (_b = (_a = proto.options) === null || _a === void 0 ? void 0 : _a.deprecated) !== null && _b !== void 0 ? _b : false,
    file,
    parent,
    name: proto.name,
    typeName: makeTypeName(proto, parent, file),
    values: [],
    sharedPrefix: findEnumSharedPrefix(proto.name, proto.value.map((v) => {
      var _a2;
      return (_a2 = v.name) !== null && _a2 !== void 0 ? _a2 : "";
    })),
    toString() {
      return `enum ${this.typeName}`;
    },
    getComments() {
      const path = this.parent ? [
        ...this.parent.getComments().sourcePath,
        FieldNumber.DescriptorProto_EnumType,
        this.parent.proto.enumType.indexOf(this.proto)
      ] : [
        FieldNumber.FileDescriptorProto_EnumType,
        this.file.proto.enumType.indexOf(this.proto)
      ];
      return findComments(file.proto.sourceCodeInfo, path);
    }
  };
  cart.enums.set(desc.typeName, desc);
  proto.value.forEach((proto4) => {
    var _a2, _b2;
    assert(proto4.name, `invalid EnumValueDescriptorProto: missing name`);
    assert(proto4.number !== void 0, `invalid EnumValueDescriptorProto: missing number`);
    desc.values.push({
      kind: "enum_value",
      proto: proto4,
      deprecated: (_b2 = (_a2 = proto4.options) === null || _a2 === void 0 ? void 0 : _a2.deprecated) !== null && _b2 !== void 0 ? _b2 : false,
      parent: desc,
      name: proto4.name,
      number: proto4.number,
      toString() {
        return `enum value ${desc.typeName}.${this.name}`;
      },
      declarationString() {
        var _a3;
        let str = `${this.name} = ${this.number}`;
        if (((_a3 = this.proto.options) === null || _a3 === void 0 ? void 0 : _a3.deprecated) === true) {
          str += " [deprecated = true]";
        }
        return str;
      },
      getComments() {
        const path = [
          ...this.parent.getComments().sourcePath,
          FieldNumber.EnumDescriptorProto_Value,
          this.parent.proto.value.indexOf(this.proto)
        ];
        return findComments(file.proto.sourceCodeInfo, path);
      }
    });
  });
  ((_c = parent === null || parent === void 0 ? void 0 : parent.nestedEnums) !== null && _c !== void 0 ? _c : file.enums).push(desc);
}
function addMessage(proto, file, parent, cart) {
  var _a, _b, _c, _d;
  assert(proto.name, `invalid DescriptorProto: missing name`);
  const desc = {
    kind: "message",
    proto,
    deprecated: (_b = (_a = proto.options) === null || _a === void 0 ? void 0 : _a.deprecated) !== null && _b !== void 0 ? _b : false,
    file,
    parent,
    name: proto.name,
    typeName: makeTypeName(proto, parent, file),
    fields: [],
    oneofs: [],
    members: [],
    nestedEnums: [],
    nestedMessages: [],
    nestedExtensions: [],
    toString() {
      return `message ${this.typeName}`;
    },
    getComments() {
      const path = this.parent ? [
        ...this.parent.getComments().sourcePath,
        FieldNumber.DescriptorProto_NestedType,
        this.parent.proto.nestedType.indexOf(this.proto)
      ] : [
        FieldNumber.FileDescriptorProto_MessageType,
        this.file.proto.messageType.indexOf(this.proto)
      ];
      return findComments(file.proto.sourceCodeInfo, path);
    }
  };
  if (((_c = proto.options) === null || _c === void 0 ? void 0 : _c.mapEntry) === true) {
    cart.mapEntries.set(desc.typeName, desc);
  } else {
    ((_d = parent === null || parent === void 0 ? void 0 : parent.nestedMessages) !== null && _d !== void 0 ? _d : file.messages).push(desc);
    cart.messages.set(desc.typeName, desc);
  }
  for (const enumProto of proto.enumType) {
    addEnum(enumProto, file, desc, cart);
  }
  for (const messageProto of proto.nestedType) {
    addMessage(messageProto, file, desc, cart);
  }
}
function addService(proto, file, cart) {
  var _a, _b;
  assert(proto.name, `invalid ServiceDescriptorProto: missing name`);
  const desc = {
    kind: "service",
    proto,
    deprecated: (_b = (_a = proto.options) === null || _a === void 0 ? void 0 : _a.deprecated) !== null && _b !== void 0 ? _b : false,
    file,
    name: proto.name,
    typeName: makeTypeName(proto, void 0, file),
    methods: [],
    toString() {
      return `service ${this.typeName}`;
    },
    getComments() {
      const path = [
        FieldNumber.FileDescriptorProto_Service,
        this.file.proto.service.indexOf(this.proto)
      ];
      return findComments(file.proto.sourceCodeInfo, path);
    }
  };
  file.services.push(desc);
  cart.services.set(desc.typeName, desc);
  for (const methodProto of proto.method) {
    desc.methods.push(newMethod(methodProto, desc, cart));
  }
}
function newMethod(proto, parent, cart) {
  var _a, _b, _c;
  assert(proto.name, `invalid MethodDescriptorProto: missing name`);
  assert(proto.inputType, `invalid MethodDescriptorProto: missing input_type`);
  assert(proto.outputType, `invalid MethodDescriptorProto: missing output_type`);
  let methodKind;
  if (proto.clientStreaming === true && proto.serverStreaming === true) {
    methodKind = MethodKind.BiDiStreaming;
  } else if (proto.clientStreaming === true) {
    methodKind = MethodKind.ClientStreaming;
  } else if (proto.serverStreaming === true) {
    methodKind = MethodKind.ServerStreaming;
  } else {
    methodKind = MethodKind.Unary;
  }
  let idempotency;
  switch ((_a = proto.options) === null || _a === void 0 ? void 0 : _a.idempotencyLevel) {
    case MethodOptions_IdempotencyLevel.IDEMPOTENT:
      idempotency = MethodIdempotency.Idempotent;
      break;
    case MethodOptions_IdempotencyLevel.NO_SIDE_EFFECTS:
      idempotency = MethodIdempotency.NoSideEffects;
      break;
    case MethodOptions_IdempotencyLevel.IDEMPOTENCY_UNKNOWN:
    case void 0:
      idempotency = void 0;
      break;
  }
  const input = cart.messages.get(trimLeadingDot(proto.inputType));
  const output = cart.messages.get(trimLeadingDot(proto.outputType));
  assert(input, `invalid MethodDescriptorProto: input_type ${proto.inputType} not found`);
  assert(output, `invalid MethodDescriptorProto: output_type ${proto.inputType} not found`);
  const name = proto.name;
  return {
    kind: "rpc",
    proto,
    deprecated: (_c = (_b = proto.options) === null || _b === void 0 ? void 0 : _b.deprecated) !== null && _c !== void 0 ? _c : false,
    parent,
    name,
    methodKind,
    input,
    output,
    idempotency,
    toString() {
      return `rpc ${parent.typeName}.${name}`;
    },
    getComments() {
      const path = [
        ...this.parent.getComments().sourcePath,
        FieldNumber.ServiceDescriptorProto_Method,
        this.parent.proto.method.indexOf(this.proto)
      ];
      return findComments(parent.file.proto.sourceCodeInfo, path);
    }
  };
}
function newOneof(proto, parent) {
  assert(proto.name, `invalid OneofDescriptorProto: missing name`);
  return {
    kind: "oneof",
    proto,
    deprecated: false,
    parent,
    fields: [],
    name: proto.name,
    toString() {
      return `oneof ${parent.typeName}.${this.name}`;
    },
    getComments() {
      const path = [
        ...this.parent.getComments().sourcePath,
        FieldNumber.DescriptorProto_OneofDecl,
        this.parent.proto.oneofDecl.indexOf(this.proto)
      ];
      return findComments(parent.file.proto.sourceCodeInfo, path);
    }
  };
}
function newField(proto, file, parent, oneof, cart) {
  var _a, _b, _c, _d;
  assert(proto.name, `invalid FieldDescriptorProto: missing name`);
  assert(proto.number, `invalid FieldDescriptorProto: missing number`);
  assert(proto.type, `invalid FieldDescriptorProto: missing type`);
  const packedByDefault = isPackedFieldByDefault(proto, file.syntax);
  const common = {
    proto,
    deprecated: (_b = (_a = proto.options) === null || _a === void 0 ? void 0 : _a.deprecated) !== null && _b !== void 0 ? _b : false,
    name: proto.name,
    number: proto.number,
    parent,
    oneof,
    optional: isOptionalField(proto, file.syntax),
    packed: (_d = (_c = proto.options) === null || _c === void 0 ? void 0 : _c.packed) !== null && _d !== void 0 ? _d : packedByDefault,
    packedByDefault,
    jsonName: proto.jsonName === fieldJsonName(proto.name) ? void 0 : proto.jsonName,
    scalar: void 0,
    message: void 0,
    enum: void 0,
    mapKey: void 0,
    mapValue: void 0,
    toString() {
      return `field ${this.parent.typeName}.${this.name}`;
    },
    declarationString,
    getComments() {
      const path = [
        ...this.parent.getComments().sourcePath,
        FieldNumber.DescriptorProto_Field,
        this.parent.proto.field.indexOf(this.proto)
      ];
      return findComments(file.proto.sourceCodeInfo, path);
    }
  };
  const repeated = proto.label === FieldDescriptorProto_Label.REPEATED;
  switch (proto.type) {
    case FieldDescriptorProto_Type.MESSAGE:
    case FieldDescriptorProto_Type.GROUP: {
      assert(proto.typeName, `invalid FieldDescriptorProto: missing type_name`);
      const mapEntry = cart.mapEntries.get(trimLeadingDot(proto.typeName));
      if (mapEntry !== void 0) {
        assert(repeated, `invalid FieldDescriptorProto: expected map entry to be repeated`);
        return Object.assign(Object.assign(Object.assign({}, common), { kind: "field", fieldKind: "map", repeated: false }), getMapFieldTypes(mapEntry));
      }
      const message = cart.messages.get(trimLeadingDot(proto.typeName));
      assert(message !== void 0, `invalid FieldDescriptorProto: type_name ${proto.typeName} not found`);
      return Object.assign(Object.assign({}, common), {
        kind: "field",
        fieldKind: "message",
        repeated,
        message
      });
    }
    case FieldDescriptorProto_Type.ENUM: {
      assert(proto.typeName, `invalid FieldDescriptorProto: missing type_name`);
      const e = cart.enums.get(trimLeadingDot(proto.typeName));
      assert(e !== void 0, `invalid FieldDescriptorProto: type_name ${proto.typeName} not found`);
      return Object.assign(Object.assign({}, common), {
        kind: "field",
        fieldKind: "enum",
        getDefaultValue,
        repeated,
        enum: e
      });
    }
    default: {
      const scalar = fieldTypeToScalarType[proto.type];
      assert(scalar, `invalid FieldDescriptorProto: unknown type ${proto.type}`);
      return Object.assign(Object.assign({}, common), {
        kind: "field",
        fieldKind: "scalar",
        getDefaultValue,
        repeated,
        scalar
      });
    }
  }
}
function newExtension(proto, file, parent, cart) {
  assert(proto.extendee, `invalid FieldDescriptorProto: missing extendee`);
  const field = newField(
    proto,
    file,
    null,
    // to safe us many lines of duplicated code, we trick the type system
    void 0,
    cart
  );
  const extendee = cart.messages.get(trimLeadingDot(proto.extendee));
  assert(extendee, `invalid FieldDescriptorProto: extendee ${proto.extendee} not found`);
  return Object.assign(Object.assign({}, field), {
    kind: "extension",
    typeName: makeTypeName(proto, parent, file),
    parent,
    file,
    extendee,
    toString() {
      return `extension ${this.typeName}`;
    },
    getComments() {
      const path = this.parent ? [
        ...this.parent.getComments().sourcePath,
        FieldNumber.DescriptorProto_Extension,
        this.parent.proto.extension.indexOf(proto)
      ] : [
        FieldNumber.FileDescriptorProto_Extension,
        this.file.proto.extension.indexOf(proto)
      ];
      return findComments(file.proto.sourceCodeInfo, path);
    }
  });
}
function makeTypeName(proto, parent, file) {
  assert(proto.name, `invalid ${proto.getType().typeName}: missing name`);
  let typeName;
  if (parent) {
    typeName = `${parent.typeName}.${proto.name}`;
  } else if (file.proto.package !== void 0) {
    typeName = `${file.proto.package}.${proto.name}`;
  } else {
    typeName = `${proto.name}`;
  }
  return typeName;
}
function trimLeadingDot(typeName) {
  return typeName.startsWith(".") ? typeName.substring(1) : typeName;
}
function getMapFieldTypes(mapEntry) {
  var _a, _b;
  assert((_a = mapEntry.proto.options) === null || _a === void 0 ? void 0 : _a.mapEntry, `invalid DescriptorProto: expected ${mapEntry.toString()} to be a map entry`);
  assert(mapEntry.fields.length === 2, `invalid DescriptorProto: map entry ${mapEntry.toString()} has ${mapEntry.fields.length} fields`);
  const keyField = mapEntry.fields.find((f) => f.proto.number === 1);
  assert(keyField, `invalid DescriptorProto: map entry ${mapEntry.toString()} is missing key field`);
  const mapKey = keyField.scalar;
  assert(mapKey !== void 0 && mapKey !== ScalarType.BYTES && mapKey !== ScalarType.FLOAT && mapKey !== ScalarType.DOUBLE, `invalid DescriptorProto: map entry ${mapEntry.toString()} has unexpected key type ${(_b = keyField.proto.type) !== null && _b !== void 0 ? _b : -1}`);
  const valueField = mapEntry.fields.find((f) => f.proto.number === 2);
  assert(valueField, `invalid DescriptorProto: map entry ${mapEntry.toString()} is missing value field`);
  switch (valueField.fieldKind) {
    case "scalar":
      return {
        mapKey,
        mapValue: Object.assign(Object.assign({}, valueField), { kind: "scalar" })
      };
    case "message":
      return {
        mapKey,
        mapValue: Object.assign(Object.assign({}, valueField), { kind: "message" })
      };
    case "enum":
      return {
        mapKey,
        mapValue: Object.assign(Object.assign({}, valueField), { kind: "enum" })
      };
    default:
      throw new Error("invalid DescriptorProto: unsupported map entry value field");
  }
}
function findOneof(proto, allOneofs) {
  var _a;
  const oneofIndex = proto.oneofIndex;
  if (oneofIndex === void 0) {
    return void 0;
  }
  let oneof;
  if (proto.proto3Optional !== true) {
    oneof = allOneofs[oneofIndex];
    assert(oneof, `invalid FieldDescriptorProto: oneof #${oneofIndex} for field #${(_a = proto.number) !== null && _a !== void 0 ? _a : -1} not found`);
  }
  return oneof;
}
function isOptionalField(proto, syntax) {
  switch (syntax) {
    case "proto2":
      return proto.oneofIndex === void 0 && proto.label === FieldDescriptorProto_Label.OPTIONAL;
    case "proto3":
      return proto.proto3Optional === true;
  }
}
function isPackedFieldByDefault(proto, syntax) {
  assert(proto.type, `invalid FieldDescriptorProto: missing type`);
  if (syntax === "proto3") {
    switch (proto.type) {
      case FieldDescriptorProto_Type.DOUBLE:
      case FieldDescriptorProto_Type.FLOAT:
      case FieldDescriptorProto_Type.INT64:
      case FieldDescriptorProto_Type.UINT64:
      case FieldDescriptorProto_Type.INT32:
      case FieldDescriptorProto_Type.FIXED64:
      case FieldDescriptorProto_Type.FIXED32:
      case FieldDescriptorProto_Type.UINT32:
      case FieldDescriptorProto_Type.SFIXED32:
      case FieldDescriptorProto_Type.SFIXED64:
      case FieldDescriptorProto_Type.SINT32:
      case FieldDescriptorProto_Type.SINT64:
      case FieldDescriptorProto_Type.BOOL:
      case FieldDescriptorProto_Type.ENUM:
        return true;
      default:
        return false;
    }
  }
  return false;
}
var fieldTypeToScalarType = {
  [FieldDescriptorProto_Type.DOUBLE]: ScalarType.DOUBLE,
  [FieldDescriptorProto_Type.FLOAT]: ScalarType.FLOAT,
  [FieldDescriptorProto_Type.INT64]: ScalarType.INT64,
  [FieldDescriptorProto_Type.UINT64]: ScalarType.UINT64,
  [FieldDescriptorProto_Type.INT32]: ScalarType.INT32,
  [FieldDescriptorProto_Type.FIXED64]: ScalarType.FIXED64,
  [FieldDescriptorProto_Type.FIXED32]: ScalarType.FIXED32,
  [FieldDescriptorProto_Type.BOOL]: ScalarType.BOOL,
  [FieldDescriptorProto_Type.STRING]: ScalarType.STRING,
  [FieldDescriptorProto_Type.GROUP]: void 0,
  [FieldDescriptorProto_Type.MESSAGE]: void 0,
  [FieldDescriptorProto_Type.BYTES]: ScalarType.BYTES,
  [FieldDescriptorProto_Type.UINT32]: ScalarType.UINT32,
  [FieldDescriptorProto_Type.ENUM]: void 0,
  [FieldDescriptorProto_Type.SFIXED32]: ScalarType.SFIXED32,
  [FieldDescriptorProto_Type.SFIXED64]: ScalarType.SFIXED64,
  [FieldDescriptorProto_Type.SINT32]: ScalarType.SINT32,
  [FieldDescriptorProto_Type.SINT64]: ScalarType.SINT64
};
function findComments(sourceCodeInfo, sourcePath) {
  if (!sourceCodeInfo) {
    return {
      leadingDetached: [],
      sourcePath
    };
  }
  for (const location of sourceCodeInfo.location) {
    if (location.path.length !== sourcePath.length) {
      continue;
    }
    if (location.path.some((value, index) => sourcePath[index] !== value)) {
      continue;
    }
    return {
      leadingDetached: location.leadingDetachedComments,
      leading: location.leadingComments,
      trailing: location.trailingComments,
      sourcePath
    };
  }
  return {
    leadingDetached: [],
    sourcePath
  };
}
var FieldNumber;
(function(FieldNumber2) {
  FieldNumber2[FieldNumber2["FileDescriptorProto_Package"] = 2] = "FileDescriptorProto_Package";
  FieldNumber2[FieldNumber2["FileDescriptorProto_MessageType"] = 4] = "FileDescriptorProto_MessageType";
  FieldNumber2[FieldNumber2["FileDescriptorProto_EnumType"] = 5] = "FileDescriptorProto_EnumType";
  FieldNumber2[FieldNumber2["FileDescriptorProto_Service"] = 6] = "FileDescriptorProto_Service";
  FieldNumber2[FieldNumber2["FileDescriptorProto_Extension"] = 7] = "FileDescriptorProto_Extension";
  FieldNumber2[FieldNumber2["FileDescriptorProto_Syntax"] = 12] = "FileDescriptorProto_Syntax";
  FieldNumber2[FieldNumber2["DescriptorProto_Field"] = 2] = "DescriptorProto_Field";
  FieldNumber2[FieldNumber2["DescriptorProto_NestedType"] = 3] = "DescriptorProto_NestedType";
  FieldNumber2[FieldNumber2["DescriptorProto_EnumType"] = 4] = "DescriptorProto_EnumType";
  FieldNumber2[FieldNumber2["DescriptorProto_Extension"] = 6] = "DescriptorProto_Extension";
  FieldNumber2[FieldNumber2["DescriptorProto_OneofDecl"] = 8] = "DescriptorProto_OneofDecl";
  FieldNumber2[FieldNumber2["EnumDescriptorProto_Value"] = 2] = "EnumDescriptorProto_Value";
  FieldNumber2[FieldNumber2["ServiceDescriptorProto_Method"] = 2] = "ServiceDescriptorProto_Method";
})(FieldNumber || (FieldNumber = {}));
function declarationString() {
  var _a, _b;
  const parts = [];
  if (this.repeated) {
    parts.push("repeated");
  }
  if (this.optional) {
    parts.push("optional");
  }
  const file = this.kind === "extension" ? this.file : this.parent.file;
  if (file.syntax == "proto2" && this.proto.label === FieldDescriptorProto_Label.REQUIRED) {
    parts.push("required");
  }
  let type;
  switch (this.fieldKind) {
    case "scalar":
      type = ScalarType[this.scalar].toLowerCase();
      break;
    case "enum":
      type = this.enum.typeName;
      break;
    case "message":
      type = this.message.typeName;
      break;
    case "map": {
      const k = ScalarType[this.mapKey].toLowerCase();
      let v;
      switch (this.mapValue.kind) {
        case "scalar":
          v = ScalarType[this.mapValue.scalar].toLowerCase();
          break;
        case "enum":
          v = this.mapValue.enum.typeName;
          break;
        case "message":
          v = this.mapValue.message.typeName;
          break;
      }
      type = `map<${k}, ${v}>`;
      break;
    }
  }
  parts.push(`${type} ${this.name} = ${this.number}`);
  const options = [];
  if (((_a = this.proto.options) === null || _a === void 0 ? void 0 : _a.packed) !== void 0) {
    options.push(`packed = ${this.proto.options.packed.toString()}`);
  }
  let defaultValue = this.proto.defaultValue;
  if (defaultValue !== void 0) {
    if (this.proto.type == FieldDescriptorProto_Type.BYTES || this.proto.type == FieldDescriptorProto_Type.STRING) {
      defaultValue = '"' + defaultValue.replace('"', '\\"') + '"';
    }
    options.push(`default = ${defaultValue}`);
  }
  if (this.jsonName !== void 0) {
    options.push(`json_name = "${this.jsonName}"`);
  }
  if (((_b = this.proto.options) === null || _b === void 0 ? void 0 : _b.deprecated) === true) {
    options.push(`deprecated = true`);
  }
  if (options.length > 0) {
    parts.push("[" + options.join(", ") + "]");
  }
  return parts.join(" ");
}
function getDefaultValue() {
  const d = this.proto.defaultValue;
  if (d === void 0) {
    return void 0;
  }
  switch (this.fieldKind) {
    case "enum": {
      const enumValue = this.enum.values.find((v) => v.name === d);
      assert(enumValue, `cannot parse ${this.toString()} default value: ${d}`);
      return enumValue.number;
    }
    case "scalar":
      switch (this.scalar) {
        case ScalarType.STRING:
          return d;
        case ScalarType.BYTES: {
          const u = unescapeBytesDefaultValue(d);
          if (u === false) {
            throw new Error(`cannot parse ${this.toString()} default value: ${d}`);
          }
          return u;
        }
        case ScalarType.INT64:
        case ScalarType.SFIXED64:
        case ScalarType.SINT64:
          return protoInt64.parse(d);
        case ScalarType.UINT64:
        case ScalarType.FIXED64:
          return protoInt64.uParse(d);
        case ScalarType.DOUBLE:
        case ScalarType.FLOAT:
          switch (d) {
            case "inf":
              return Number.POSITIVE_INFINITY;
            case "-inf":
              return Number.NEGATIVE_INFINITY;
            case "nan":
              return Number.NaN;
            default:
              return parseFloat(d);
          }
        case ScalarType.BOOL:
          return d === "true";
        case ScalarType.INT32:
        case ScalarType.UINT32:
        case ScalarType.SINT32:
        case ScalarType.FIXED32:
        case ScalarType.SFIXED32:
          return parseInt(d, 10);
      }
      break;
    default:
      return void 0;
  }
}
function unescapeBytesDefaultValue(str) {
  const b = [];
  const input = {
    tail: str,
    c: "",
    next() {
      if (this.tail.length == 0) {
        return false;
      }
      this.c = this.tail[0];
      this.tail = this.tail.substring(1);
      return true;
    },
    take(n) {
      if (this.tail.length >= n) {
        const r = this.tail.substring(0, n);
        this.tail = this.tail.substring(n);
        return r;
      }
      return false;
    }
  };
  while (input.next()) {
    switch (input.c) {
      case "\\":
        if (input.next()) {
          switch (input.c) {
            case "\\":
              b.push(input.c.charCodeAt(0));
              break;
            case "b":
              b.push(8);
              break;
            case "f":
              b.push(12);
              break;
            case "n":
              b.push(10);
              break;
            case "r":
              b.push(13);
              break;
            case "t":
              b.push(9);
              break;
            case "v":
              b.push(11);
              break;
            case "0":
            case "1":
            case "2":
            case "3":
            case "4":
            case "5":
            case "6":
            case "7": {
              const s = input.c;
              const t = input.take(2);
              if (t === false) {
                return false;
              }
              const n = parseInt(s + t, 8);
              if (isNaN(n)) {
                return false;
              }
              b.push(n);
              break;
            }
            case "x": {
              const s = input.c;
              const t = input.take(2);
              if (t === false) {
                return false;
              }
              const n = parseInt(s + t, 16);
              if (isNaN(n)) {
                return false;
              }
              b.push(n);
              break;
            }
            case "u": {
              const s = input.c;
              const t = input.take(4);
              if (t === false) {
                return false;
              }
              const n = parseInt(s + t, 16);
              if (isNaN(n)) {
                return false;
              }
              const chunk = new Uint8Array(4);
              const view = new DataView(chunk.buffer);
              view.setInt32(0, n, true);
              b.push(chunk[0], chunk[1], chunk[2], chunk[3]);
              break;
            }
            case "U": {
              const s = input.c;
              const t = input.take(8);
              if (t === false) {
                return false;
              }
              const tc = protoInt64.uEnc(s + t);
              const chunk = new Uint8Array(8);
              const view = new DataView(chunk.buffer);
              view.setInt32(0, tc.lo, true);
              view.setInt32(4, tc.hi, true);
              b.push(chunk[0], chunk[1], chunk[2], chunk[3], chunk[4], chunk[5], chunk[6], chunk[7]);
              break;
            }
          }
        }
        break;
      default:
        b.push(input.c.charCodeAt(0));
    }
  }
  return new Uint8Array(b);
}

// node_modules/@bufbuild/protobuf/dist/esm/create-registry.js
function createRegistry(...types) {
  const messages = {};
  const enums = {};
  const services = {};
  const registry = {
    /**
     * Add a type to the registry. For messages, the types used in message
     * fields are added recursively. For services, the message types used
     * for requests and responses are added recursively.
     */
    add(type) {
      if ("fields" in type) {
        if (!this.findMessage(type.typeName)) {
          messages[type.typeName] = type;
          for (const field of type.fields.list()) {
            if (field.kind == "message") {
              this.add(field.T);
            } else if (field.kind == "map" && field.V.kind == "message") {
              this.add(field.V.T);
            } else if (field.kind == "enum") {
              this.add(field.T);
            }
          }
        }
      } else if ("methods" in type) {
        if (!this.findService(type.typeName)) {
          services[type.typeName] = type;
          for (const method of Object.values(type.methods)) {
            this.add(method.I);
            this.add(method.O);
          }
        }
      } else {
        enums[type.typeName] = type;
      }
    },
    findMessage(typeName) {
      return messages[typeName];
    },
    findEnum(typeName) {
      return enums[typeName];
    },
    findService(typeName) {
      return services[typeName];
    }
  };
  for (const type of types) {
    registry.add(type);
  }
  return registry;
}

// node_modules/@bufbuild/protobuf/dist/esm/google/protobuf/timestamp_pb.js
var Timestamp = class _Timestamp extends Message {
  constructor(data) {
    super();
    this.seconds = protoInt64.zero;
    this.nanos = 0;
    proto3.util.initPartial(data, this);
  }
  fromJson(json, options) {
    if (typeof json !== "string") {
      throw new Error(`cannot decode google.protobuf.Timestamp from JSON: ${proto3.json.debug(json)}`);
    }
    const matches = json.match(/^([0-9]{4})-([0-9]{2})-([0-9]{2})T([0-9]{2}):([0-9]{2}):([0-9]{2})(?:Z|\.([0-9]{3,9})Z|([+-][0-9][0-9]:[0-9][0-9]))$/);
    if (!matches) {
      throw new Error(`cannot decode google.protobuf.Timestamp from JSON: invalid RFC 3339 string`);
    }
    const ms = Date.parse(matches[1] + "-" + matches[2] + "-" + matches[3] + "T" + matches[4] + ":" + matches[5] + ":" + matches[6] + (matches[8] ? matches[8] : "Z"));
    if (Number.isNaN(ms)) {
      throw new Error(`cannot decode google.protobuf.Timestamp from JSON: invalid RFC 3339 string`);
    }
    if (ms < Date.parse("0001-01-01T00:00:00Z") || ms > Date.parse("9999-12-31T23:59:59Z")) {
      throw new Error(`cannot decode message google.protobuf.Timestamp from JSON: must be from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z inclusive`);
    }
    this.seconds = protoInt64.parse(ms / 1e3);
    this.nanos = 0;
    if (matches[7]) {
      this.nanos = parseInt("1" + matches[7] + "0".repeat(9 - matches[7].length)) - 1e9;
    }
    return this;
  }
  toJson(options) {
    const ms = Number(this.seconds) * 1e3;
    if (ms < Date.parse("0001-01-01T00:00:00Z") || ms > Date.parse("9999-12-31T23:59:59Z")) {
      throw new Error(`cannot encode google.protobuf.Timestamp to JSON: must be from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z inclusive`);
    }
    if (this.nanos < 0) {
      throw new Error(`cannot encode google.protobuf.Timestamp to JSON: nanos must not be negative`);
    }
    let z = "Z";
    if (this.nanos > 0) {
      const nanosStr = (this.nanos + 1e9).toString().substring(1);
      if (nanosStr.substring(3) === "000000") {
        z = "." + nanosStr.substring(0, 3) + "Z";
      } else if (nanosStr.substring(6) === "000") {
        z = "." + nanosStr.substring(0, 6) + "Z";
      } else {
        z = "." + nanosStr + "Z";
      }
    }
    return new Date(ms).toISOString().replace(".000Z", z);
  }
  toDate() {
    return new Date(Number(this.seconds) * 1e3 + Math.ceil(this.nanos / 1e6));
  }
  static now() {
    return _Timestamp.fromDate(/* @__PURE__ */ new Date());
  }
  static fromDate(date) {
    const ms = date.getTime();
    return new _Timestamp({
      seconds: protoInt64.parse(Math.floor(ms / 1e3)),
      nanos: ms % 1e3 * 1e6
    });
  }
  static fromBinary(bytes, options) {
    return new _Timestamp().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _Timestamp().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _Timestamp().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_Timestamp, a, b);
  }
};
Timestamp.runtime = proto3;
Timestamp.typeName = "google.protobuf.Timestamp";
Timestamp.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "seconds",
    kind: "scalar",
    T: 3
    /* ScalarType.INT64 */
  },
  {
    no: 2,
    name: "nanos",
    kind: "scalar",
    T: 5
    /* ScalarType.INT32 */
  }
]);

// node_modules/@bufbuild/protobuf/dist/esm/google/protobuf/duration_pb.js
var Duration = class _Duration extends Message {
  constructor(data) {
    super();
    this.seconds = protoInt64.zero;
    this.nanos = 0;
    proto3.util.initPartial(data, this);
  }
  fromJson(json, options) {
    if (typeof json !== "string") {
      throw new Error(`cannot decode google.protobuf.Duration from JSON: ${proto3.json.debug(json)}`);
    }
    const match = json.match(/^(-?[0-9]+)(?:\.([0-9]+))?s/);
    if (match === null) {
      throw new Error(`cannot decode google.protobuf.Duration from JSON: ${proto3.json.debug(json)}`);
    }
    const longSeconds = Number(match[1]);
    if (longSeconds > 315576e6 || longSeconds < -315576e6) {
      throw new Error(`cannot decode google.protobuf.Duration from JSON: ${proto3.json.debug(json)}`);
    }
    this.seconds = protoInt64.parse(longSeconds);
    if (typeof match[2] == "string") {
      const nanosStr = match[2] + "0".repeat(9 - match[2].length);
      this.nanos = parseInt(nanosStr);
      if (longSeconds < 0 || Object.is(longSeconds, -0)) {
        this.nanos = -this.nanos;
      }
    }
    return this;
  }
  toJson(options) {
    if (Number(this.seconds) > 315576e6 || Number(this.seconds) < -315576e6) {
      throw new Error(`cannot encode google.protobuf.Duration to JSON: value out of range`);
    }
    let text = this.seconds.toString();
    if (this.nanos !== 0) {
      let nanosStr = Math.abs(this.nanos).toString();
      nanosStr = "0".repeat(9 - nanosStr.length) + nanosStr;
      if (nanosStr.substring(3) === "000000") {
        nanosStr = nanosStr.substring(0, 3);
      } else if (nanosStr.substring(6) === "000") {
        nanosStr = nanosStr.substring(0, 6);
      }
      text += "." + nanosStr;
      if (this.nanos < 0 && this.seconds === protoInt64.zero) {
        text = "-" + text;
      }
    }
    return text + "s";
  }
  static fromBinary(bytes, options) {
    return new _Duration().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _Duration().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _Duration().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_Duration, a, b);
  }
};
Duration.runtime = proto3;
Duration.typeName = "google.protobuf.Duration";
Duration.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "seconds",
    kind: "scalar",
    T: 3
    /* ScalarType.INT64 */
  },
  {
    no: 2,
    name: "nanos",
    kind: "scalar",
    T: 5
    /* ScalarType.INT32 */
  }
]);

// node_modules/@bufbuild/protobuf/dist/esm/google/protobuf/any_pb.js
var Any = class _Any extends Message {
  constructor(data) {
    super();
    this.typeUrl = "";
    this.value = new Uint8Array(0);
    proto3.util.initPartial(data, this);
  }
  toJson(options) {
    var _a;
    if (this.typeUrl === "") {
      return {};
    }
    const typeName = this.typeUrlToName(this.typeUrl);
    const messageType = (_a = options === null || options === void 0 ? void 0 : options.typeRegistry) === null || _a === void 0 ? void 0 : _a.findMessage(typeName);
    if (!messageType) {
      throw new Error(`cannot encode message google.protobuf.Any to JSON: "${this.typeUrl}" is not in the type registry`);
    }
    const message = messageType.fromBinary(this.value);
    let json = message.toJson(options);
    if (typeName.startsWith("google.protobuf.") || (json === null || Array.isArray(json) || typeof json !== "object")) {
      json = { value: json };
    }
    json["@type"] = this.typeUrl;
    return json;
  }
  fromJson(json, options) {
    var _a;
    if (json === null || Array.isArray(json) || typeof json != "object") {
      throw new Error(`cannot decode message google.protobuf.Any from JSON: expected object but got ${json === null ? "null" : Array.isArray(json) ? "array" : typeof json}`);
    }
    if (Object.keys(json).length == 0) {
      return this;
    }
    const typeUrl = json["@type"];
    if (typeof typeUrl != "string" || typeUrl == "") {
      throw new Error(`cannot decode message google.protobuf.Any from JSON: "@type" is empty`);
    }
    const typeName = this.typeUrlToName(typeUrl), messageType = (_a = options === null || options === void 0 ? void 0 : options.typeRegistry) === null || _a === void 0 ? void 0 : _a.findMessage(typeName);
    if (!messageType) {
      throw new Error(`cannot decode message google.protobuf.Any from JSON: ${typeUrl} is not in the type registry`);
    }
    let message;
    if (typeName.startsWith("google.protobuf.") && Object.prototype.hasOwnProperty.call(json, "value")) {
      message = messageType.fromJson(json["value"], options);
    } else {
      const copy = Object.assign({}, json);
      delete copy["@type"];
      message = messageType.fromJson(copy, options);
    }
    this.packFrom(message);
    return this;
  }
  packFrom(message) {
    this.value = message.toBinary();
    this.typeUrl = this.typeNameToUrl(message.getType().typeName);
  }
  unpackTo(target) {
    if (!this.is(target.getType())) {
      return false;
    }
    target.fromBinary(this.value);
    return true;
  }
  unpack(registry) {
    if (this.typeUrl === "") {
      return void 0;
    }
    const messageType = registry.findMessage(this.typeUrlToName(this.typeUrl));
    if (!messageType) {
      return void 0;
    }
    return messageType.fromBinary(this.value);
  }
  is(type) {
    if (this.typeUrl === "") {
      return false;
    }
    const name = this.typeUrlToName(this.typeUrl);
    let typeName = "";
    if (typeof type === "string") {
      typeName = type;
    } else {
      typeName = type.typeName;
    }
    return name === typeName;
  }
  typeNameToUrl(name) {
    return `type.googleapis.com/${name}`;
  }
  typeUrlToName(url) {
    if (!url.length) {
      throw new Error(`invalid type url: ${url}`);
    }
    const slash = url.lastIndexOf("/");
    const name = slash > 0 ? url.substring(slash + 1) : url;
    if (!name.length) {
      throw new Error(`invalid type url: ${url}`);
    }
    return name;
  }
  static pack(message) {
    const any = new _Any();
    any.packFrom(message);
    return any;
  }
  static fromBinary(bytes, options) {
    return new _Any().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _Any().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _Any().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_Any, a, b);
  }
};
Any.runtime = proto3;
Any.typeName = "google.protobuf.Any";
Any.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "type_url",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  },
  {
    no: 2,
    name: "value",
    kind: "scalar",
    T: 12
    /* ScalarType.BYTES */
  }
]);

// node_modules/@bufbuild/protobuf/dist/esm/google/protobuf/empty_pb.js
var Empty = class _Empty extends Message {
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _Empty().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _Empty().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _Empty().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_Empty, a, b);
  }
};
Empty.runtime = proto3;
Empty.typeName = "google.protobuf.Empty";
Empty.fields = proto3.util.newFieldList(() => []);

// node_modules/@bufbuild/protobuf/dist/esm/google/protobuf/field_mask_pb.js
var FieldMask = class _FieldMask extends Message {
  constructor(data) {
    super();
    this.paths = [];
    proto3.util.initPartial(data, this);
  }
  toJson(options) {
    function protoCamelCase2(snakeCase) {
      let capNext = false;
      const b = [];
      for (let i = 0; i < snakeCase.length; i++) {
        let c = snakeCase.charAt(i);
        switch (c) {
          case "_":
            capNext = true;
            break;
          case "0":
          case "1":
          case "2":
          case "3":
          case "4":
          case "5":
          case "6":
          case "7":
          case "8":
          case "9":
            b.push(c);
            capNext = false;
            break;
          default:
            if (capNext) {
              capNext = false;
              c = c.toUpperCase();
            }
            b.push(c);
            break;
        }
      }
      return b.join("");
    }
    return this.paths.map((p) => {
      if (p.match(/_[0-9]?_/g) || p.match(/[A-Z]/g)) {
        throw new Error('cannot encode google.protobuf.FieldMask to JSON: lowerCamelCase of path name "' + p + '" is irreversible');
      }
      return protoCamelCase2(p);
    }).join(",");
  }
  fromJson(json, options) {
    if (typeof json !== "string") {
      throw new Error("cannot decode google.protobuf.FieldMask from JSON: " + proto3.json.debug(json));
    }
    if (json === "") {
      return this;
    }
    function camelToSnake(str) {
      if (str.includes("_")) {
        throw new Error("cannot decode google.protobuf.FieldMask from JSON: path names must be lowerCamelCase");
      }
      const sc = str.replace(/[A-Z]/g, (letter) => "_" + letter.toLowerCase());
      return sc[0] === "_" ? sc.substring(1) : sc;
    }
    this.paths = json.split(",").map(camelToSnake);
    return this;
  }
  static fromBinary(bytes, options) {
    return new _FieldMask().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _FieldMask().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _FieldMask().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_FieldMask, a, b);
  }
};
FieldMask.runtime = proto3;
FieldMask.typeName = "google.protobuf.FieldMask";
FieldMask.fields = proto3.util.newFieldList(() => [
  { no: 1, name: "paths", kind: "scalar", T: 9, repeated: true }
]);

// node_modules/@bufbuild/protobuf/dist/esm/google/protobuf/struct_pb.js
var NullValue;
(function(NullValue2) {
  NullValue2[NullValue2["NULL_VALUE"] = 0] = "NULL_VALUE";
})(NullValue || (NullValue = {}));
proto3.util.setEnumType(NullValue, "google.protobuf.NullValue", [
  { no: 0, name: "NULL_VALUE" }
]);
var Struct = class _Struct extends Message {
  constructor(data) {
    super();
    this.fields = {};
    proto3.util.initPartial(data, this);
  }
  toJson(options) {
    const json = {};
    for (const [k, v] of Object.entries(this.fields)) {
      json[k] = v.toJson(options);
    }
    return json;
  }
  fromJson(json, options) {
    if (typeof json != "object" || json == null || Array.isArray(json)) {
      throw new Error("cannot decode google.protobuf.Struct from JSON " + proto3.json.debug(json));
    }
    for (const [k, v] of Object.entries(json)) {
      this.fields[k] = Value.fromJson(v);
    }
    return this;
  }
  static fromBinary(bytes, options) {
    return new _Struct().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _Struct().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _Struct().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_Struct, a, b);
  }
};
Struct.runtime = proto3;
Struct.typeName = "google.protobuf.Struct";
Struct.fields = proto3.util.newFieldList(() => [
  { no: 1, name: "fields", kind: "map", K: 9, V: { kind: "message", T: Value } }
]);
var Value = class _Value extends Message {
  constructor(data) {
    super();
    this.kind = { case: void 0 };
    proto3.util.initPartial(data, this);
  }
  toJson(options) {
    switch (this.kind.case) {
      case "nullValue":
        return null;
      case "numberValue":
        if (!Number.isFinite(this.kind.value)) {
          throw new Error("google.protobuf.Value cannot be NaN or Infinity");
        }
        return this.kind.value;
      case "boolValue":
        return this.kind.value;
      case "stringValue":
        return this.kind.value;
      case "structValue":
      case "listValue":
        return this.kind.value.toJson(Object.assign(Object.assign({}, options), { emitDefaultValues: true }));
    }
    throw new Error("google.protobuf.Value must have a value");
  }
  fromJson(json, options) {
    switch (typeof json) {
      case "number":
        this.kind = { case: "numberValue", value: json };
        break;
      case "string":
        this.kind = { case: "stringValue", value: json };
        break;
      case "boolean":
        this.kind = { case: "boolValue", value: json };
        break;
      case "object":
        if (json === null) {
          this.kind = { case: "nullValue", value: NullValue.NULL_VALUE };
        } else if (Array.isArray(json)) {
          this.kind = { case: "listValue", value: ListValue.fromJson(json) };
        } else {
          this.kind = { case: "structValue", value: Struct.fromJson(json) };
        }
        break;
      default:
        throw new Error("cannot decode google.protobuf.Value from JSON " + proto3.json.debug(json));
    }
    return this;
  }
  static fromBinary(bytes, options) {
    return new _Value().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _Value().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _Value().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_Value, a, b);
  }
};
Value.runtime = proto3;
Value.typeName = "google.protobuf.Value";
Value.fields = proto3.util.newFieldList(() => [
  { no: 1, name: "null_value", kind: "enum", T: proto3.getEnumType(NullValue), oneof: "kind" },
  { no: 2, name: "number_value", kind: "scalar", T: 1, oneof: "kind" },
  { no: 3, name: "string_value", kind: "scalar", T: 9, oneof: "kind" },
  { no: 4, name: "bool_value", kind: "scalar", T: 8, oneof: "kind" },
  { no: 5, name: "struct_value", kind: "message", T: Struct, oneof: "kind" },
  { no: 6, name: "list_value", kind: "message", T: ListValue, oneof: "kind" }
]);
var ListValue = class _ListValue extends Message {
  constructor(data) {
    super();
    this.values = [];
    proto3.util.initPartial(data, this);
  }
  toJson(options) {
    return this.values.map((v) => v.toJson());
  }
  fromJson(json, options) {
    if (!Array.isArray(json)) {
      throw new Error("cannot decode google.protobuf.ListValue from JSON " + proto3.json.debug(json));
    }
    for (let e of json) {
      this.values.push(Value.fromJson(e));
    }
    return this;
  }
  static fromBinary(bytes, options) {
    return new _ListValue().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _ListValue().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _ListValue().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_ListValue, a, b);
  }
};
ListValue.runtime = proto3;
ListValue.typeName = "google.protobuf.ListValue";
ListValue.fields = proto3.util.newFieldList(() => [
  { no: 1, name: "values", kind: "message", T: Value, repeated: true }
]);

// node_modules/@bufbuild/protobuf/dist/esm/google/protobuf/wrappers_pb.js
var DoubleValue = class _DoubleValue extends Message {
  constructor(data) {
    super();
    this.value = 0;
    proto3.util.initPartial(data, this);
  }
  toJson(options) {
    return proto3.json.writeScalar(ScalarType.DOUBLE, this.value, true);
  }
  fromJson(json, options) {
    try {
      this.value = proto3.json.readScalar(ScalarType.DOUBLE, json);
    } catch (e) {
      let m = `cannot decode message google.protobuf.DoubleValue from JSON"`;
      if (e instanceof Error && e.message.length > 0) {
        m += `: ${e.message}`;
      }
      throw new Error(m);
    }
    return this;
  }
  static fromBinary(bytes, options) {
    return new _DoubleValue().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _DoubleValue().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _DoubleValue().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_DoubleValue, a, b);
  }
};
DoubleValue.runtime = proto3;
DoubleValue.typeName = "google.protobuf.DoubleValue";
DoubleValue.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "value",
    kind: "scalar",
    T: 1
    /* ScalarType.DOUBLE */
  }
]);
DoubleValue.fieldWrapper = {
  wrapField(value) {
    return new DoubleValue({ value });
  },
  unwrapField(value) {
    return value.value;
  }
};
var FloatValue = class _FloatValue extends Message {
  constructor(data) {
    super();
    this.value = 0;
    proto3.util.initPartial(data, this);
  }
  toJson(options) {
    return proto3.json.writeScalar(ScalarType.FLOAT, this.value, true);
  }
  fromJson(json, options) {
    try {
      this.value = proto3.json.readScalar(ScalarType.FLOAT, json);
    } catch (e) {
      let m = `cannot decode message google.protobuf.FloatValue from JSON"`;
      if (e instanceof Error && e.message.length > 0) {
        m += `: ${e.message}`;
      }
      throw new Error(m);
    }
    return this;
  }
  static fromBinary(bytes, options) {
    return new _FloatValue().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _FloatValue().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _FloatValue().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_FloatValue, a, b);
  }
};
FloatValue.runtime = proto3;
FloatValue.typeName = "google.protobuf.FloatValue";
FloatValue.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "value",
    kind: "scalar",
    T: 2
    /* ScalarType.FLOAT */
  }
]);
FloatValue.fieldWrapper = {
  wrapField(value) {
    return new FloatValue({ value });
  },
  unwrapField(value) {
    return value.value;
  }
};
var Int64Value = class _Int64Value extends Message {
  constructor(data) {
    super();
    this.value = protoInt64.zero;
    proto3.util.initPartial(data, this);
  }
  toJson(options) {
    return proto3.json.writeScalar(ScalarType.INT64, this.value, true);
  }
  fromJson(json, options) {
    try {
      this.value = proto3.json.readScalar(ScalarType.INT64, json);
    } catch (e) {
      let m = `cannot decode message google.protobuf.Int64Value from JSON"`;
      if (e instanceof Error && e.message.length > 0) {
        m += `: ${e.message}`;
      }
      throw new Error(m);
    }
    return this;
  }
  static fromBinary(bytes, options) {
    return new _Int64Value().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _Int64Value().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _Int64Value().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_Int64Value, a, b);
  }
};
Int64Value.runtime = proto3;
Int64Value.typeName = "google.protobuf.Int64Value";
Int64Value.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "value",
    kind: "scalar",
    T: 3
    /* ScalarType.INT64 */
  }
]);
Int64Value.fieldWrapper = {
  wrapField(value) {
    return new Int64Value({ value });
  },
  unwrapField(value) {
    return value.value;
  }
};
var UInt64Value = class _UInt64Value extends Message {
  constructor(data) {
    super();
    this.value = protoInt64.zero;
    proto3.util.initPartial(data, this);
  }
  toJson(options) {
    return proto3.json.writeScalar(ScalarType.UINT64, this.value, true);
  }
  fromJson(json, options) {
    try {
      this.value = proto3.json.readScalar(ScalarType.UINT64, json);
    } catch (e) {
      let m = `cannot decode message google.protobuf.UInt64Value from JSON"`;
      if (e instanceof Error && e.message.length > 0) {
        m += `: ${e.message}`;
      }
      throw new Error(m);
    }
    return this;
  }
  static fromBinary(bytes, options) {
    return new _UInt64Value().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _UInt64Value().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _UInt64Value().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_UInt64Value, a, b);
  }
};
UInt64Value.runtime = proto3;
UInt64Value.typeName = "google.protobuf.UInt64Value";
UInt64Value.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "value",
    kind: "scalar",
    T: 4
    /* ScalarType.UINT64 */
  }
]);
UInt64Value.fieldWrapper = {
  wrapField(value) {
    return new UInt64Value({ value });
  },
  unwrapField(value) {
    return value.value;
  }
};
var Int32Value = class _Int32Value extends Message {
  constructor(data) {
    super();
    this.value = 0;
    proto3.util.initPartial(data, this);
  }
  toJson(options) {
    return proto3.json.writeScalar(ScalarType.INT32, this.value, true);
  }
  fromJson(json, options) {
    try {
      this.value = proto3.json.readScalar(ScalarType.INT32, json);
    } catch (e) {
      let m = `cannot decode message google.protobuf.Int32Value from JSON"`;
      if (e instanceof Error && e.message.length > 0) {
        m += `: ${e.message}`;
      }
      throw new Error(m);
    }
    return this;
  }
  static fromBinary(bytes, options) {
    return new _Int32Value().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _Int32Value().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _Int32Value().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_Int32Value, a, b);
  }
};
Int32Value.runtime = proto3;
Int32Value.typeName = "google.protobuf.Int32Value";
Int32Value.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "value",
    kind: "scalar",
    T: 5
    /* ScalarType.INT32 */
  }
]);
Int32Value.fieldWrapper = {
  wrapField(value) {
    return new Int32Value({ value });
  },
  unwrapField(value) {
    return value.value;
  }
};
var UInt32Value = class _UInt32Value extends Message {
  constructor(data) {
    super();
    this.value = 0;
    proto3.util.initPartial(data, this);
  }
  toJson(options) {
    return proto3.json.writeScalar(ScalarType.UINT32, this.value, true);
  }
  fromJson(json, options) {
    try {
      this.value = proto3.json.readScalar(ScalarType.UINT32, json);
    } catch (e) {
      let m = `cannot decode message google.protobuf.UInt32Value from JSON"`;
      if (e instanceof Error && e.message.length > 0) {
        m += `: ${e.message}`;
      }
      throw new Error(m);
    }
    return this;
  }
  static fromBinary(bytes, options) {
    return new _UInt32Value().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _UInt32Value().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _UInt32Value().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_UInt32Value, a, b);
  }
};
UInt32Value.runtime = proto3;
UInt32Value.typeName = "google.protobuf.UInt32Value";
UInt32Value.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "value",
    kind: "scalar",
    T: 13
    /* ScalarType.UINT32 */
  }
]);
UInt32Value.fieldWrapper = {
  wrapField(value) {
    return new UInt32Value({ value });
  },
  unwrapField(value) {
    return value.value;
  }
};
var BoolValue = class _BoolValue extends Message {
  constructor(data) {
    super();
    this.value = false;
    proto3.util.initPartial(data, this);
  }
  toJson(options) {
    return proto3.json.writeScalar(ScalarType.BOOL, this.value, true);
  }
  fromJson(json, options) {
    try {
      this.value = proto3.json.readScalar(ScalarType.BOOL, json);
    } catch (e) {
      let m = `cannot decode message google.protobuf.BoolValue from JSON"`;
      if (e instanceof Error && e.message.length > 0) {
        m += `: ${e.message}`;
      }
      throw new Error(m);
    }
    return this;
  }
  static fromBinary(bytes, options) {
    return new _BoolValue().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _BoolValue().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _BoolValue().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_BoolValue, a, b);
  }
};
BoolValue.runtime = proto3;
BoolValue.typeName = "google.protobuf.BoolValue";
BoolValue.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "value",
    kind: "scalar",
    T: 8
    /* ScalarType.BOOL */
  }
]);
BoolValue.fieldWrapper = {
  wrapField(value) {
    return new BoolValue({ value });
  },
  unwrapField(value) {
    return value.value;
  }
};
var StringValue = class _StringValue extends Message {
  constructor(data) {
    super();
    this.value = "";
    proto3.util.initPartial(data, this);
  }
  toJson(options) {
    return proto3.json.writeScalar(ScalarType.STRING, this.value, true);
  }
  fromJson(json, options) {
    try {
      this.value = proto3.json.readScalar(ScalarType.STRING, json);
    } catch (e) {
      let m = `cannot decode message google.protobuf.StringValue from JSON"`;
      if (e instanceof Error && e.message.length > 0) {
        m += `: ${e.message}`;
      }
      throw new Error(m);
    }
    return this;
  }
  static fromBinary(bytes, options) {
    return new _StringValue().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _StringValue().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _StringValue().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_StringValue, a, b);
  }
};
StringValue.runtime = proto3;
StringValue.typeName = "google.protobuf.StringValue";
StringValue.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "value",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  }
]);
StringValue.fieldWrapper = {
  wrapField(value) {
    return new StringValue({ value });
  },
  unwrapField(value) {
    return value.value;
  }
};
var BytesValue = class _BytesValue extends Message {
  constructor(data) {
    super();
    this.value = new Uint8Array(0);
    proto3.util.initPartial(data, this);
  }
  toJson(options) {
    return proto3.json.writeScalar(ScalarType.BYTES, this.value, true);
  }
  fromJson(json, options) {
    try {
      this.value = proto3.json.readScalar(ScalarType.BYTES, json);
    } catch (e) {
      let m = `cannot decode message google.protobuf.BytesValue from JSON"`;
      if (e instanceof Error && e.message.length > 0) {
        m += `: ${e.message}`;
      }
      throw new Error(m);
    }
    return this;
  }
  static fromBinary(bytes, options) {
    return new _BytesValue().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _BytesValue().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _BytesValue().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_BytesValue, a, b);
  }
};
BytesValue.runtime = proto3;
BytesValue.typeName = "google.protobuf.BytesValue";
BytesValue.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "value",
    kind: "scalar",
    T: 12
    /* ScalarType.BYTES */
  }
]);
BytesValue.fieldWrapper = {
  wrapField(value) {
    return new BytesValue({ value });
  },
  unwrapField(value) {
    return value.value;
  }
};

// node_modules/@bufbuild/protobuf/dist/esm/create-registry-from-desc.js
var wkMessages = [
  Any,
  Duration,
  Empty,
  FieldMask,
  Struct,
  Value,
  ListValue,
  Timestamp,
  Duration,
  DoubleValue,
  FloatValue,
  Int64Value,
  Int32Value,
  UInt32Value,
  UInt64Value,
  BoolValue,
  StringValue,
  BytesValue
];
var wkEnums = [getEnumType(NullValue)];
function createRegistryFromDescriptors(input, replaceWkt = true) {
  const set = input instanceof Uint8Array || input instanceof FileDescriptorSet ? createDescriptorSet(input) : input;
  const enums = {};
  const messages = {};
  const services = {};
  if (replaceWkt) {
    for (const mt of wkMessages) {
      messages[mt.typeName] = mt;
    }
    for (const et of wkEnums) {
      enums[et.typeName] = et;
    }
  }
  return {
    /**
     * May raise an error on invalid descriptors.
     */
    findEnum(typeName) {
      const existing = enums[typeName];
      if (existing) {
        return existing;
      }
      const desc = set.enums.get(typeName);
      if (!desc) {
        return void 0;
      }
      const runtime = desc.file.syntax == "proto3" ? proto3 : proto2;
      const type = runtime.makeEnumType(typeName, desc.values.map((u) => ({
        no: u.number,
        name: u.name,
        localName: localName(u)
      })), {});
      enums[typeName] = type;
      return type;
    },
    /**
     * May raise an error on invalid descriptors.
     */
    findMessage(typeName) {
      const existing = messages[typeName];
      if (existing) {
        return existing;
      }
      const desc = set.messages.get(typeName);
      if (!desc) {
        return void 0;
      }
      const runtime = desc.file.syntax == "proto3" ? proto3 : proto2;
      const fields = [];
      const type = runtime.makeMessageType(typeName, () => fields, {
        localName: localName(desc)
      });
      messages[typeName] = type;
      for (const field of desc.fields) {
        const fieldInfo = makeFieldInfo(field, this);
        fields.push(fieldInfo);
      }
      return type;
    },
    /**
     * May raise an error on invalid descriptors.
     */
    findService(typeName) {
      const existing = services[typeName];
      if (existing) {
        return existing;
      }
      const desc = set.services.get(typeName);
      if (!desc) {
        return void 0;
      }
      const methods = {};
      for (const method of desc.methods) {
        const I = this.findMessage(method.input.typeName);
        const O = this.findMessage(method.output.typeName);
        assert(I, `message "${method.input.typeName}" for ${method.toString()} not found`);
        assert(O, `output message "${method.output.typeName}" for ${method.toString()} not found`);
        methods[localName(method)] = {
          name: method.name,
          I,
          O,
          kind: method.methodKind,
          idempotency: method.idempotency
          // We do not surface options at this time
          // options: {},
        };
      }
      return services[typeName] = {
        typeName: desc.typeName,
        methods
      };
    }
  };
}
function makeFieldInfo(desc, resolver) {
  switch (desc.fieldKind) {
    case "map":
      return makeMapFieldInfo(desc, resolver);
    case "message":
      return makeMessageFieldInfo(desc, resolver);
    case "enum": {
      const fi = makeEnumFieldInfo(desc, resolver);
      fi.default = desc.getDefaultValue();
      return fi;
    }
    case "scalar": {
      const fi = makeScalarFieldInfo(desc);
      fi.default = desc.getDefaultValue();
      return fi;
    }
  }
}
function makeMapFieldInfo(field, resolver) {
  const base = {
    kind: "map",
    no: field.number,
    name: field.name,
    jsonName: field.jsonName,
    K: field.mapKey
  };
  if (field.mapValue.message) {
    const messageType = resolver.findMessage(field.mapValue.message.typeName);
    assert(messageType, `message "${field.mapValue.message.typeName}" for ${field.toString()} not found`);
    return Object.assign(Object.assign({}, base), { V: {
      kind: "message",
      T: messageType
    } });
  }
  if (field.mapValue.enum) {
    const enumType = resolver.findEnum(field.mapValue.enum.typeName);
    assert(enumType, `enum "${field.mapValue.enum.typeName}" for ${field.toString()} not found`);
    return Object.assign(Object.assign({}, base), { V: {
      kind: "enum",
      T: enumType
    } });
  }
  return Object.assign(Object.assign({}, base), { V: {
    kind: "scalar",
    T: field.mapValue.scalar
  } });
}
function makeScalarFieldInfo(field) {
  const base = {
    kind: "scalar",
    no: field.number,
    name: field.name,
    jsonName: field.jsonName,
    T: field.scalar
  };
  if (field.repeated) {
    return Object.assign(Object.assign({}, base), { repeated: true, packed: field.packed, oneof: void 0, T: field.scalar });
  }
  if (field.oneof) {
    return Object.assign(Object.assign({}, base), { oneof: field.oneof.name });
  }
  if (field.optional) {
    return Object.assign(Object.assign({}, base), { opt: true });
  }
  return base;
}
function makeMessageFieldInfo(field, resolver) {
  const messageType = resolver.findMessage(field.message.typeName);
  assert(messageType, `message "${field.message.typeName}" for ${field.toString()} not found`);
  const base = {
    kind: "message",
    no: field.number,
    name: field.name,
    jsonName: field.jsonName,
    T: messageType
  };
  if (field.repeated) {
    return Object.assign(Object.assign({}, base), { repeated: true, packed: field.packed, oneof: void 0 });
  }
  if (field.oneof) {
    return Object.assign(Object.assign({}, base), { oneof: field.oneof.name });
  }
  if (field.optional) {
    return Object.assign(Object.assign({}, base), { opt: true });
  }
  return base;
}
function makeEnumFieldInfo(field, resolver) {
  const enumType = resolver.findEnum(field.enum.typeName);
  assert(enumType, `enum "${field.enum.typeName}" for ${field.toString()} not found`);
  const base = {
    kind: "enum",
    no: field.number,
    name: field.name,
    jsonName: field.jsonName,
    T: enumType
  };
  if (field.repeated) {
    return Object.assign(Object.assign({}, base), { repeated: true, packed: field.packed, oneof: void 0 });
  }
  if (field.oneof) {
    return Object.assign(Object.assign({}, base), { oneof: field.oneof.name });
  }
  if (field.optional) {
    return Object.assign(Object.assign({}, base), { opt: true });
  }
  return base;
}

// node_modules/@bufbuild/protobuf/dist/esm/to-plain-message.js
function toPlainMessage(message) {
  const type = message.getType();
  const target = {};
  for (const member of type.fields.byMember()) {
    const source = message[member.localName];
    let copy;
    if (member.repeated) {
      copy = source.map((e) => toPlainValue(e));
    } else if (member.kind == "map") {
      copy = {};
      for (const [key, v] of Object.entries(source)) {
        copy[key] = toPlainValue(v);
      }
    } else if (member.kind == "oneof") {
      const f = member.findField(source.case);
      copy = f ? { case: source.case, value: toPlainValue(source.value) } : { case: void 0 };
    } else {
      copy = toPlainValue(source);
    }
    target[member.localName] = copy;
  }
  return target;
}
function toPlainValue(value) {
  if (value === void 0) {
    return value;
  }
  if (value instanceof Message) {
    return toPlainMessage(value);
  }
  if (value instanceof Uint8Array) {
    const c = new Uint8Array(value.byteLength);
    c.set(value);
    return c;
  }
  return value;
}

// node_modules/@bufbuild/protobuf/dist/esm/google/protobuf/compiler/plugin_pb.js
var Version = class _Version extends Message {
  constructor(data) {
    super();
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _Version().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _Version().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _Version().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_Version, a, b);
  }
};
Version.runtime = proto2;
Version.typeName = "google.protobuf.compiler.Version";
Version.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "major", kind: "scalar", T: 5, opt: true },
  { no: 2, name: "minor", kind: "scalar", T: 5, opt: true },
  { no: 3, name: "patch", kind: "scalar", T: 5, opt: true },
  { no: 4, name: "suffix", kind: "scalar", T: 9, opt: true }
]);
var CodeGeneratorRequest = class _CodeGeneratorRequest extends Message {
  constructor(data) {
    super();
    this.fileToGenerate = [];
    this.protoFile = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _CodeGeneratorRequest().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _CodeGeneratorRequest().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _CodeGeneratorRequest().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_CodeGeneratorRequest, a, b);
  }
};
CodeGeneratorRequest.runtime = proto2;
CodeGeneratorRequest.typeName = "google.protobuf.compiler.CodeGeneratorRequest";
CodeGeneratorRequest.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "file_to_generate", kind: "scalar", T: 9, repeated: true },
  { no: 2, name: "parameter", kind: "scalar", T: 9, opt: true },
  { no: 15, name: "proto_file", kind: "message", T: FileDescriptorProto, repeated: true },
  { no: 3, name: "compiler_version", kind: "message", T: Version, opt: true }
]);
var CodeGeneratorResponse = class _CodeGeneratorResponse extends Message {
  constructor(data) {
    super();
    this.file = [];
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _CodeGeneratorResponse().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _CodeGeneratorResponse().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _CodeGeneratorResponse().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_CodeGeneratorResponse, a, b);
  }
};
CodeGeneratorResponse.runtime = proto2;
CodeGeneratorResponse.typeName = "google.protobuf.compiler.CodeGeneratorResponse";
CodeGeneratorResponse.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "error", kind: "scalar", T: 9, opt: true },
  { no: 2, name: "supported_features", kind: "scalar", T: 4, opt: true },
  { no: 15, name: "file", kind: "message", T: CodeGeneratorResponse_File, repeated: true }
]);
var CodeGeneratorResponse_Feature;
(function(CodeGeneratorResponse_Feature2) {
  CodeGeneratorResponse_Feature2[CodeGeneratorResponse_Feature2["NONE"] = 0] = "NONE";
  CodeGeneratorResponse_Feature2[CodeGeneratorResponse_Feature2["PROTO3_OPTIONAL"] = 1] = "PROTO3_OPTIONAL";
})(CodeGeneratorResponse_Feature || (CodeGeneratorResponse_Feature = {}));
proto2.util.setEnumType(CodeGeneratorResponse_Feature, "google.protobuf.compiler.CodeGeneratorResponse.Feature", [
  { no: 0, name: "FEATURE_NONE" },
  { no: 1, name: "FEATURE_PROTO3_OPTIONAL" }
]);
var CodeGeneratorResponse_File = class _CodeGeneratorResponse_File extends Message {
  constructor(data) {
    super();
    proto2.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _CodeGeneratorResponse_File().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _CodeGeneratorResponse_File().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _CodeGeneratorResponse_File().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto2.util.equals(_CodeGeneratorResponse_File, a, b);
  }
};
CodeGeneratorResponse_File.runtime = proto2;
CodeGeneratorResponse_File.typeName = "google.protobuf.compiler.CodeGeneratorResponse.File";
CodeGeneratorResponse_File.fields = proto2.util.newFieldList(() => [
  { no: 1, name: "name", kind: "scalar", T: 9, opt: true },
  { no: 2, name: "insertion_point", kind: "scalar", T: 9, opt: true },
  { no: 15, name: "content", kind: "scalar", T: 9, opt: true },
  { no: 16, name: "generated_code_info", kind: "message", T: GeneratedCodeInfo, opt: true }
]);

// node_modules/@bufbuild/protobuf/dist/esm/google/protobuf/source_context_pb.js
var SourceContext = class _SourceContext extends Message {
  constructor(data) {
    super();
    this.fileName = "";
    proto3.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _SourceContext().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _SourceContext().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _SourceContext().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_SourceContext, a, b);
  }
};
SourceContext.runtime = proto3;
SourceContext.typeName = "google.protobuf.SourceContext";
SourceContext.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "file_name",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  }
]);

// node_modules/@bufbuild/protobuf/dist/esm/google/protobuf/type_pb.js
var Syntax;
(function(Syntax2) {
  Syntax2[Syntax2["PROTO2"] = 0] = "PROTO2";
  Syntax2[Syntax2["PROTO3"] = 1] = "PROTO3";
  Syntax2[Syntax2["EDITIONS"] = 2] = "EDITIONS";
})(Syntax || (Syntax = {}));
proto3.util.setEnumType(Syntax, "google.protobuf.Syntax", [
  { no: 0, name: "SYNTAX_PROTO2" },
  { no: 1, name: "SYNTAX_PROTO3" },
  { no: 2, name: "SYNTAX_EDITIONS" }
]);
var Type = class _Type extends Message {
  constructor(data) {
    super();
    this.name = "";
    this.fields = [];
    this.oneofs = [];
    this.options = [];
    this.syntax = Syntax.PROTO2;
    this.edition = "";
    proto3.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _Type().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _Type().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _Type().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_Type, a, b);
  }
};
Type.runtime = proto3;
Type.typeName = "google.protobuf.Type";
Type.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "name",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  },
  { no: 2, name: "fields", kind: "message", T: Field, repeated: true },
  { no: 3, name: "oneofs", kind: "scalar", T: 9, repeated: true },
  { no: 4, name: "options", kind: "message", T: Option, repeated: true },
  { no: 5, name: "source_context", kind: "message", T: SourceContext },
  { no: 6, name: "syntax", kind: "enum", T: proto3.getEnumType(Syntax) },
  {
    no: 7,
    name: "edition",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  }
]);
var Field = class _Field extends Message {
  constructor(data) {
    super();
    this.kind = Field_Kind.TYPE_UNKNOWN;
    this.cardinality = Field_Cardinality.UNKNOWN;
    this.number = 0;
    this.name = "";
    this.typeUrl = "";
    this.oneofIndex = 0;
    this.packed = false;
    this.options = [];
    this.jsonName = "";
    this.defaultValue = "";
    proto3.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _Field().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _Field().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _Field().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_Field, a, b);
  }
};
Field.runtime = proto3;
Field.typeName = "google.protobuf.Field";
Field.fields = proto3.util.newFieldList(() => [
  { no: 1, name: "kind", kind: "enum", T: proto3.getEnumType(Field_Kind) },
  { no: 2, name: "cardinality", kind: "enum", T: proto3.getEnumType(Field_Cardinality) },
  {
    no: 3,
    name: "number",
    kind: "scalar",
    T: 5
    /* ScalarType.INT32 */
  },
  {
    no: 4,
    name: "name",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  },
  {
    no: 6,
    name: "type_url",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  },
  {
    no: 7,
    name: "oneof_index",
    kind: "scalar",
    T: 5
    /* ScalarType.INT32 */
  },
  {
    no: 8,
    name: "packed",
    kind: "scalar",
    T: 8
    /* ScalarType.BOOL */
  },
  { no: 9, name: "options", kind: "message", T: Option, repeated: true },
  {
    no: 10,
    name: "json_name",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  },
  {
    no: 11,
    name: "default_value",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  }
]);
var Field_Kind;
(function(Field_Kind2) {
  Field_Kind2[Field_Kind2["TYPE_UNKNOWN"] = 0] = "TYPE_UNKNOWN";
  Field_Kind2[Field_Kind2["TYPE_DOUBLE"] = 1] = "TYPE_DOUBLE";
  Field_Kind2[Field_Kind2["TYPE_FLOAT"] = 2] = "TYPE_FLOAT";
  Field_Kind2[Field_Kind2["TYPE_INT64"] = 3] = "TYPE_INT64";
  Field_Kind2[Field_Kind2["TYPE_UINT64"] = 4] = "TYPE_UINT64";
  Field_Kind2[Field_Kind2["TYPE_INT32"] = 5] = "TYPE_INT32";
  Field_Kind2[Field_Kind2["TYPE_FIXED64"] = 6] = "TYPE_FIXED64";
  Field_Kind2[Field_Kind2["TYPE_FIXED32"] = 7] = "TYPE_FIXED32";
  Field_Kind2[Field_Kind2["TYPE_BOOL"] = 8] = "TYPE_BOOL";
  Field_Kind2[Field_Kind2["TYPE_STRING"] = 9] = "TYPE_STRING";
  Field_Kind2[Field_Kind2["TYPE_GROUP"] = 10] = "TYPE_GROUP";
  Field_Kind2[Field_Kind2["TYPE_MESSAGE"] = 11] = "TYPE_MESSAGE";
  Field_Kind2[Field_Kind2["TYPE_BYTES"] = 12] = "TYPE_BYTES";
  Field_Kind2[Field_Kind2["TYPE_UINT32"] = 13] = "TYPE_UINT32";
  Field_Kind2[Field_Kind2["TYPE_ENUM"] = 14] = "TYPE_ENUM";
  Field_Kind2[Field_Kind2["TYPE_SFIXED32"] = 15] = "TYPE_SFIXED32";
  Field_Kind2[Field_Kind2["TYPE_SFIXED64"] = 16] = "TYPE_SFIXED64";
  Field_Kind2[Field_Kind2["TYPE_SINT32"] = 17] = "TYPE_SINT32";
  Field_Kind2[Field_Kind2["TYPE_SINT64"] = 18] = "TYPE_SINT64";
})(Field_Kind || (Field_Kind = {}));
proto3.util.setEnumType(Field_Kind, "google.protobuf.Field.Kind", [
  { no: 0, name: "TYPE_UNKNOWN" },
  { no: 1, name: "TYPE_DOUBLE" },
  { no: 2, name: "TYPE_FLOAT" },
  { no: 3, name: "TYPE_INT64" },
  { no: 4, name: "TYPE_UINT64" },
  { no: 5, name: "TYPE_INT32" },
  { no: 6, name: "TYPE_FIXED64" },
  { no: 7, name: "TYPE_FIXED32" },
  { no: 8, name: "TYPE_BOOL" },
  { no: 9, name: "TYPE_STRING" },
  { no: 10, name: "TYPE_GROUP" },
  { no: 11, name: "TYPE_MESSAGE" },
  { no: 12, name: "TYPE_BYTES" },
  { no: 13, name: "TYPE_UINT32" },
  { no: 14, name: "TYPE_ENUM" },
  { no: 15, name: "TYPE_SFIXED32" },
  { no: 16, name: "TYPE_SFIXED64" },
  { no: 17, name: "TYPE_SINT32" },
  { no: 18, name: "TYPE_SINT64" }
]);
var Field_Cardinality;
(function(Field_Cardinality2) {
  Field_Cardinality2[Field_Cardinality2["UNKNOWN"] = 0] = "UNKNOWN";
  Field_Cardinality2[Field_Cardinality2["OPTIONAL"] = 1] = "OPTIONAL";
  Field_Cardinality2[Field_Cardinality2["REQUIRED"] = 2] = "REQUIRED";
  Field_Cardinality2[Field_Cardinality2["REPEATED"] = 3] = "REPEATED";
})(Field_Cardinality || (Field_Cardinality = {}));
proto3.util.setEnumType(Field_Cardinality, "google.protobuf.Field.Cardinality", [
  { no: 0, name: "CARDINALITY_UNKNOWN" },
  { no: 1, name: "CARDINALITY_OPTIONAL" },
  { no: 2, name: "CARDINALITY_REQUIRED" },
  { no: 3, name: "CARDINALITY_REPEATED" }
]);
var Enum = class _Enum extends Message {
  constructor(data) {
    super();
    this.name = "";
    this.enumvalue = [];
    this.options = [];
    this.syntax = Syntax.PROTO2;
    this.edition = "";
    proto3.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _Enum().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _Enum().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _Enum().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_Enum, a, b);
  }
};
Enum.runtime = proto3;
Enum.typeName = "google.protobuf.Enum";
Enum.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "name",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  },
  { no: 2, name: "enumvalue", kind: "message", T: EnumValue, repeated: true },
  { no: 3, name: "options", kind: "message", T: Option, repeated: true },
  { no: 4, name: "source_context", kind: "message", T: SourceContext },
  { no: 5, name: "syntax", kind: "enum", T: proto3.getEnumType(Syntax) },
  {
    no: 6,
    name: "edition",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  }
]);
var EnumValue = class _EnumValue extends Message {
  constructor(data) {
    super();
    this.name = "";
    this.number = 0;
    this.options = [];
    proto3.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _EnumValue().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _EnumValue().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _EnumValue().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_EnumValue, a, b);
  }
};
EnumValue.runtime = proto3;
EnumValue.typeName = "google.protobuf.EnumValue";
EnumValue.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "name",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  },
  {
    no: 2,
    name: "number",
    kind: "scalar",
    T: 5
    /* ScalarType.INT32 */
  },
  { no: 3, name: "options", kind: "message", T: Option, repeated: true }
]);
var Option = class _Option extends Message {
  constructor(data) {
    super();
    this.name = "";
    proto3.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _Option().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _Option().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _Option().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_Option, a, b);
  }
};
Option.runtime = proto3;
Option.typeName = "google.protobuf.Option";
Option.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "name",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  },
  { no: 2, name: "value", kind: "message", T: Any }
]);

// node_modules/@bufbuild/protobuf/dist/esm/google/protobuf/api_pb.js
var Api = class _Api extends Message {
  constructor(data) {
    super();
    this.name = "";
    this.methods = [];
    this.options = [];
    this.version = "";
    this.mixins = [];
    this.syntax = Syntax.PROTO2;
    proto3.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _Api().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _Api().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _Api().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_Api, a, b);
  }
};
Api.runtime = proto3;
Api.typeName = "google.protobuf.Api";
Api.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "name",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  },
  { no: 2, name: "methods", kind: "message", T: Method, repeated: true },
  { no: 3, name: "options", kind: "message", T: Option, repeated: true },
  {
    no: 4,
    name: "version",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  },
  { no: 5, name: "source_context", kind: "message", T: SourceContext },
  { no: 6, name: "mixins", kind: "message", T: Mixin, repeated: true },
  { no: 7, name: "syntax", kind: "enum", T: proto3.getEnumType(Syntax) }
]);
var Method = class _Method extends Message {
  constructor(data) {
    super();
    this.name = "";
    this.requestTypeUrl = "";
    this.requestStreaming = false;
    this.responseTypeUrl = "";
    this.responseStreaming = false;
    this.options = [];
    this.syntax = Syntax.PROTO2;
    proto3.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _Method().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _Method().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _Method().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_Method, a, b);
  }
};
Method.runtime = proto3;
Method.typeName = "google.protobuf.Method";
Method.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "name",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  },
  {
    no: 2,
    name: "request_type_url",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  },
  {
    no: 3,
    name: "request_streaming",
    kind: "scalar",
    T: 8
    /* ScalarType.BOOL */
  },
  {
    no: 4,
    name: "response_type_url",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  },
  {
    no: 5,
    name: "response_streaming",
    kind: "scalar",
    T: 8
    /* ScalarType.BOOL */
  },
  { no: 6, name: "options", kind: "message", T: Option, repeated: true },
  { no: 7, name: "syntax", kind: "enum", T: proto3.getEnumType(Syntax) }
]);
var Mixin = class _Mixin extends Message {
  constructor(data) {
    super();
    this.name = "";
    this.root = "";
    proto3.util.initPartial(data, this);
  }
  static fromBinary(bytes, options) {
    return new _Mixin().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new _Mixin().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new _Mixin().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(_Mixin, a, b);
  }
};
Mixin.runtime = proto3;
Mixin.typeName = "google.protobuf.Mixin";
Mixin.fields = proto3.util.newFieldList(() => [
  {
    no: 1,
    name: "name",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  },
  {
    no: 2,
    name: "root",
    kind: "scalar",
    T: 9
    /* ScalarType.STRING */
  }
]);

export {
  Message,
  ScalarType,
  protoInt64,
  WireType,
  BinaryWriter,
  BinaryReader,
  protoBase64,
  proto3,
  proto2,
  protoDouble,
  protoDelimited,
  codegenInfo,
  MethodKind,
  MethodIdempotency,
  FileDescriptorSet,
  FileDescriptorProto,
  DescriptorProto,
  DescriptorProto_ExtensionRange,
  DescriptorProto_ReservedRange,
  ExtensionRangeOptions,
  ExtensionRangeOptions_VerificationState,
  ExtensionRangeOptions_Declaration,
  FieldDescriptorProto,
  FieldDescriptorProto_Type,
  FieldDescriptorProto_Label,
  OneofDescriptorProto,
  EnumDescriptorProto,
  EnumDescriptorProto_EnumReservedRange,
  EnumValueDescriptorProto,
  ServiceDescriptorProto,
  MethodDescriptorProto,
  FileOptions,
  FileOptions_OptimizeMode,
  MessageOptions,
  FieldOptions,
  FieldOptions_CType,
  FieldOptions_JSType,
  FieldOptions_OptionRetention,
  FieldOptions_OptionTargetType,
  OneofOptions,
  EnumOptions,
  EnumValueOptions,
  ServiceOptions,
  MethodOptions,
  MethodOptions_IdempotencyLevel,
  UninterpretedOption,
  UninterpretedOption_NamePart,
  SourceCodeInfo,
  SourceCodeInfo_Location,
  GeneratedCodeInfo,
  GeneratedCodeInfo_Annotation,
  GeneratedCodeInfo_Annotation_Semantic,
  createDescriptorSet,
  createRegistry,
  Timestamp,
  Duration,
  Any,
  Empty,
  FieldMask,
  NullValue,
  Struct,
  Value,
  ListValue,
  DoubleValue,
  FloatValue,
  Int64Value,
  UInt64Value,
  Int32Value,
  UInt32Value,
  BoolValue,
  StringValue,
  BytesValue,
  createRegistryFromDescriptors,
  toPlainMessage,
  Version,
  CodeGeneratorRequest,
  CodeGeneratorResponse,
  CodeGeneratorResponse_Feature,
  CodeGeneratorResponse_File,
  SourceContext,
  Syntax,
  Type,
  Field,
  Field_Kind,
  Field_Cardinality,
  Enum,
  EnumValue,
  Option,
  Api,
  Method,
  Mixin
};
//# sourceMappingURL=chunk-TNC6V2Y3.js.map
