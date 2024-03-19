/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal.js";

export const protobufPackage = "filter";

/**
 * StringFilter matches the value of a string against a set of rules.
 * All of the non-zero rules must match for the filter to match.
 * An empty filter matches any.
 */
export interface StringFilter {
  /** Empty matches the value against the empty value. */
  empty: boolean;
  /** NotEmpty matches the value against a not-empty value. */
  notEmpty: boolean;
  /** Value matches an exact value. */
  value: string;
  /**
   * Values matches one or more exact values.
   * If any of the values match, this field is considered matched.
   */
  values: string[];
  /** Re matches the value against a regular expression. */
  re: string;
  /** HasPrefix checks if the value has the given prefix. */
  hasPrefix: string;
  /** HasSuffix checks if the value has the given suffix. */
  hasSuffix: string;
  /** Contains checks if the value contains the given value. */
  contains: string;
}

function createBaseStringFilter(): StringFilter {
  return { empty: false, notEmpty: false, value: "", values: [], re: "", hasPrefix: "", hasSuffix: "", contains: "" };
}

export const StringFilter = {
  encode(message: StringFilter, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.empty !== false) {
      writer.uint32(8).bool(message.empty);
    }
    if (message.notEmpty !== false) {
      writer.uint32(16).bool(message.notEmpty);
    }
    if (message.value !== "") {
      writer.uint32(26).string(message.value);
    }
    for (const v of message.values) {
      writer.uint32(34).string(v!);
    }
    if (message.re !== "") {
      writer.uint32(42).string(message.re);
    }
    if (message.hasPrefix !== "") {
      writer.uint32(50).string(message.hasPrefix);
    }
    if (message.hasSuffix !== "") {
      writer.uint32(58).string(message.hasSuffix);
    }
    if (message.contains !== "") {
      writer.uint32(66).string(message.contains);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): StringFilter {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStringFilter();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.empty = reader.bool();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.notEmpty = reader.bool();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.value = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.values.push(reader.string());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.re = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.hasPrefix = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.hasSuffix = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.contains = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  // encodeTransform encodes a source of message objects.
  // Transform<StringFilter, Uint8Array>
  async *encodeTransform(
    source: AsyncIterable<StringFilter | StringFilter[]> | Iterable<StringFilter | StringFilter[]>,
  ): AsyncIterable<Uint8Array> {
    for await (const pkt of source) {
      if (globalThis.Array.isArray(pkt)) {
        for (const p of (pkt as any)) {
          yield* [StringFilter.encode(p).finish()];
        }
      } else {
        yield* [StringFilter.encode(pkt as any).finish()];
      }
    }
  },

  // decodeTransform decodes a source of encoded messages.
  // Transform<Uint8Array, StringFilter>
  async *decodeTransform(
    source: AsyncIterable<Uint8Array | Uint8Array[]> | Iterable<Uint8Array | Uint8Array[]>,
  ): AsyncIterable<StringFilter> {
    for await (const pkt of source) {
      if (globalThis.Array.isArray(pkt)) {
        for (const p of (pkt as any)) {
          yield* [StringFilter.decode(p)];
        }
      } else {
        yield* [StringFilter.decode(pkt as any)];
      }
    }
  },

  fromJSON(object: any): StringFilter {
    return {
      empty: isSet(object.empty) ? globalThis.Boolean(object.empty) : false,
      notEmpty: isSet(object.notEmpty) ? globalThis.Boolean(object.notEmpty) : false,
      value: isSet(object.value) ? globalThis.String(object.value) : "",
      values: globalThis.Array.isArray(object?.values) ? object.values.map((e: any) => globalThis.String(e)) : [],
      re: isSet(object.re) ? globalThis.String(object.re) : "",
      hasPrefix: isSet(object.hasPrefix) ? globalThis.String(object.hasPrefix) : "",
      hasSuffix: isSet(object.hasSuffix) ? globalThis.String(object.hasSuffix) : "",
      contains: isSet(object.contains) ? globalThis.String(object.contains) : "",
    };
  },

  toJSON(message: StringFilter): unknown {
    const obj: any = {};
    if (message.empty !== false) {
      obj.empty = message.empty;
    }
    if (message.notEmpty !== false) {
      obj.notEmpty = message.notEmpty;
    }
    if (message.value !== "") {
      obj.value = message.value;
    }
    if (message.values?.length) {
      obj.values = message.values;
    }
    if (message.re !== "") {
      obj.re = message.re;
    }
    if (message.hasPrefix !== "") {
      obj.hasPrefix = message.hasPrefix;
    }
    if (message.hasSuffix !== "") {
      obj.hasSuffix = message.hasSuffix;
    }
    if (message.contains !== "") {
      obj.contains = message.contains;
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<StringFilter>, I>>(base?: I): StringFilter {
    return StringFilter.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<StringFilter>, I>>(object: I): StringFilter {
    const message = createBaseStringFilter();
    message.empty = object.empty ?? false;
    message.notEmpty = object.notEmpty ?? false;
    message.value = object.value ?? "";
    message.values = object.values?.map((e) => e) || [];
    message.re = object.re ?? "";
    message.hasPrefix = object.hasPrefix ?? "";
    message.hasSuffix = object.hasSuffix ?? "";
    message.contains = object.contains ?? "";
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Long ? string | number | Long : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends { $case: string } ? { [K in keyof Omit<T, "$case">]?: DeepPartial<T[K]> } & { $case: T["$case"] }
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

type KeysOfUnion<T> = T extends T ? keyof T : never;
export type Exact<P, I extends P> = P extends Builtin ? P
  : P & { [K in keyof P]: Exact<P[K], I[K]> } & { [K in Exclude<keyof I, KeysOfUnion<P>>]: never };

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
