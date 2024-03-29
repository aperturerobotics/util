/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal.js";

export const protobufPackage = "backoff";

/** BackoffKind is the kind of backoff. */
export enum BackoffKind {
  /** BackoffKind_UNKNOWN - BackoffKind_UNKNOWN defaults to BackoffKind_EXPONENTIAL */
  BackoffKind_UNKNOWN = 0,
  /** BackoffKind_EXPONENTIAL - BackoffKind_EXPONENTIAL is an exponential backoff. */
  BackoffKind_EXPONENTIAL = 1,
  /** BackoffKind_CONSTANT - BackoffKind_CONSTANT is a constant backoff. */
  BackoffKind_CONSTANT = 2,
  UNRECOGNIZED = -1,
}

export function backoffKindFromJSON(object: any): BackoffKind {
  switch (object) {
    case 0:
    case "BackoffKind_UNKNOWN":
      return BackoffKind.BackoffKind_UNKNOWN;
    case 1:
    case "BackoffKind_EXPONENTIAL":
      return BackoffKind.BackoffKind_EXPONENTIAL;
    case 2:
    case "BackoffKind_CONSTANT":
      return BackoffKind.BackoffKind_CONSTANT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return BackoffKind.UNRECOGNIZED;
  }
}

export function backoffKindToJSON(object: BackoffKind): string {
  switch (object) {
    case BackoffKind.BackoffKind_UNKNOWN:
      return "BackoffKind_UNKNOWN";
    case BackoffKind.BackoffKind_EXPONENTIAL:
      return "BackoffKind_EXPONENTIAL";
    case BackoffKind.BackoffKind_CONSTANT:
      return "BackoffKind_CONSTANT";
    case BackoffKind.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/** Backoff configures a backoff. */
export interface Backoff {
  /** BackoffKind is the kind of backoff. */
  backoffKind: BackoffKind;
  /** Exponential is the arguments for an exponential backoff. */
  exponential:
    | Exponential
    | undefined;
  /** Constant is the arugment for a constant backoff. */
  constant: Constant | undefined;
}

/** Exponential is the exponential arguments. */
export interface Exponential {
  /**
   * InitialInterval is the initial interval in milliseconds.
   * Default: 800ms.
   */
  initialInterval: number;
  /**
   * Multiplier is the timing multiplier.
   * Default: 1.8
   */
  multiplier: number;
  /**
   * MaxInterval is the maximum timing interval in milliseconds.
   * Default: 20 seconds
   */
  maxInterval: number;
  /**
   * RandomizationFactor is the randomization factor.
   * Should be from [0, 1] as a percentage of the retry interval.
   *
   * randomized interval = RetryInterval * (random value in range [1 - RandomizationFactor, 1 + RandomizationFactor])
   *
   * Default: 0 (disabled)
   */
  randomizationFactor: number;
  /**
   * MaxElapsedTime if set specifies a maximum time for the backoff, in milliseconds.
   * After this time the backoff and attached process terminates.
   * May be empty, might be ignored.
   */
  maxElapsedTime: number;
}

/** Constant contains constant backoff options. */
export interface Constant {
  /**
   * Interval is the timing to back off, in milliseconds.
   * Defaults to 5 seconds.
   */
  interval: number;
}

function createBaseBackoff(): Backoff {
  return { backoffKind: 0, exponential: undefined, constant: undefined };
}

export const Backoff = {
  encode(message: Backoff, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.backoffKind !== 0) {
      writer.uint32(8).int32(message.backoffKind);
    }
    if (message.exponential !== undefined) {
      Exponential.encode(message.exponential, writer.uint32(18).fork()).ldelim();
    }
    if (message.constant !== undefined) {
      Constant.encode(message.constant, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Backoff {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBackoff();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.backoffKind = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.exponential = Exponential.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.constant = Constant.decode(reader, reader.uint32());
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
  // Transform<Backoff, Uint8Array>
  async *encodeTransform(
    source: AsyncIterable<Backoff | Backoff[]> | Iterable<Backoff | Backoff[]>,
  ): AsyncIterable<Uint8Array> {
    for await (const pkt of source) {
      if (globalThis.Array.isArray(pkt)) {
        for (const p of (pkt as any)) {
          yield* [Backoff.encode(p).finish()];
        }
      } else {
        yield* [Backoff.encode(pkt as any).finish()];
      }
    }
  },

  // decodeTransform decodes a source of encoded messages.
  // Transform<Uint8Array, Backoff>
  async *decodeTransform(
    source: AsyncIterable<Uint8Array | Uint8Array[]> | Iterable<Uint8Array | Uint8Array[]>,
  ): AsyncIterable<Backoff> {
    for await (const pkt of source) {
      if (globalThis.Array.isArray(pkt)) {
        for (const p of (pkt as any)) {
          yield* [Backoff.decode(p)];
        }
      } else {
        yield* [Backoff.decode(pkt as any)];
      }
    }
  },

  fromJSON(object: any): Backoff {
    return {
      backoffKind: isSet(object.backoffKind) ? backoffKindFromJSON(object.backoffKind) : 0,
      exponential: isSet(object.exponential) ? Exponential.fromJSON(object.exponential) : undefined,
      constant: isSet(object.constant) ? Constant.fromJSON(object.constant) : undefined,
    };
  },

  toJSON(message: Backoff): unknown {
    const obj: any = {};
    if (message.backoffKind !== 0) {
      obj.backoffKind = backoffKindToJSON(message.backoffKind);
    }
    if (message.exponential !== undefined) {
      obj.exponential = Exponential.toJSON(message.exponential);
    }
    if (message.constant !== undefined) {
      obj.constant = Constant.toJSON(message.constant);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Backoff>, I>>(base?: I): Backoff {
    return Backoff.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Backoff>, I>>(object: I): Backoff {
    const message = createBaseBackoff();
    message.backoffKind = object.backoffKind ?? 0;
    message.exponential = (object.exponential !== undefined && object.exponential !== null)
      ? Exponential.fromPartial(object.exponential)
      : undefined;
    message.constant = (object.constant !== undefined && object.constant !== null)
      ? Constant.fromPartial(object.constant)
      : undefined;
    return message;
  },
};

function createBaseExponential(): Exponential {
  return { initialInterval: 0, multiplier: 0, maxInterval: 0, randomizationFactor: 0, maxElapsedTime: 0 };
}

export const Exponential = {
  encode(message: Exponential, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.initialInterval !== 0) {
      writer.uint32(8).uint32(message.initialInterval);
    }
    if (message.multiplier !== 0) {
      writer.uint32(21).float(message.multiplier);
    }
    if (message.maxInterval !== 0) {
      writer.uint32(24).uint32(message.maxInterval);
    }
    if (message.randomizationFactor !== 0) {
      writer.uint32(37).float(message.randomizationFactor);
    }
    if (message.maxElapsedTime !== 0) {
      writer.uint32(40).uint32(message.maxElapsedTime);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Exponential {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExponential();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.initialInterval = reader.uint32();
          continue;
        case 2:
          if (tag !== 21) {
            break;
          }

          message.multiplier = reader.float();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.maxInterval = reader.uint32();
          continue;
        case 4:
          if (tag !== 37) {
            break;
          }

          message.randomizationFactor = reader.float();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.maxElapsedTime = reader.uint32();
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
  // Transform<Exponential, Uint8Array>
  async *encodeTransform(
    source: AsyncIterable<Exponential | Exponential[]> | Iterable<Exponential | Exponential[]>,
  ): AsyncIterable<Uint8Array> {
    for await (const pkt of source) {
      if (globalThis.Array.isArray(pkt)) {
        for (const p of (pkt as any)) {
          yield* [Exponential.encode(p).finish()];
        }
      } else {
        yield* [Exponential.encode(pkt as any).finish()];
      }
    }
  },

  // decodeTransform decodes a source of encoded messages.
  // Transform<Uint8Array, Exponential>
  async *decodeTransform(
    source: AsyncIterable<Uint8Array | Uint8Array[]> | Iterable<Uint8Array | Uint8Array[]>,
  ): AsyncIterable<Exponential> {
    for await (const pkt of source) {
      if (globalThis.Array.isArray(pkt)) {
        for (const p of (pkt as any)) {
          yield* [Exponential.decode(p)];
        }
      } else {
        yield* [Exponential.decode(pkt as any)];
      }
    }
  },

  fromJSON(object: any): Exponential {
    return {
      initialInterval: isSet(object.initialInterval) ? globalThis.Number(object.initialInterval) : 0,
      multiplier: isSet(object.multiplier) ? globalThis.Number(object.multiplier) : 0,
      maxInterval: isSet(object.maxInterval) ? globalThis.Number(object.maxInterval) : 0,
      randomizationFactor: isSet(object.randomizationFactor) ? globalThis.Number(object.randomizationFactor) : 0,
      maxElapsedTime: isSet(object.maxElapsedTime) ? globalThis.Number(object.maxElapsedTime) : 0,
    };
  },

  toJSON(message: Exponential): unknown {
    const obj: any = {};
    if (message.initialInterval !== 0) {
      obj.initialInterval = Math.round(message.initialInterval);
    }
    if (message.multiplier !== 0) {
      obj.multiplier = message.multiplier;
    }
    if (message.maxInterval !== 0) {
      obj.maxInterval = Math.round(message.maxInterval);
    }
    if (message.randomizationFactor !== 0) {
      obj.randomizationFactor = message.randomizationFactor;
    }
    if (message.maxElapsedTime !== 0) {
      obj.maxElapsedTime = Math.round(message.maxElapsedTime);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Exponential>, I>>(base?: I): Exponential {
    return Exponential.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Exponential>, I>>(object: I): Exponential {
    const message = createBaseExponential();
    message.initialInterval = object.initialInterval ?? 0;
    message.multiplier = object.multiplier ?? 0;
    message.maxInterval = object.maxInterval ?? 0;
    message.randomizationFactor = object.randomizationFactor ?? 0;
    message.maxElapsedTime = object.maxElapsedTime ?? 0;
    return message;
  },
};

function createBaseConstant(): Constant {
  return { interval: 0 };
}

export const Constant = {
  encode(message: Constant, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.interval !== 0) {
      writer.uint32(8).uint32(message.interval);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Constant {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseConstant();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.interval = reader.uint32();
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
  // Transform<Constant, Uint8Array>
  async *encodeTransform(
    source: AsyncIterable<Constant | Constant[]> | Iterable<Constant | Constant[]>,
  ): AsyncIterable<Uint8Array> {
    for await (const pkt of source) {
      if (globalThis.Array.isArray(pkt)) {
        for (const p of (pkt as any)) {
          yield* [Constant.encode(p).finish()];
        }
      } else {
        yield* [Constant.encode(pkt as any).finish()];
      }
    }
  },

  // decodeTransform decodes a source of encoded messages.
  // Transform<Uint8Array, Constant>
  async *decodeTransform(
    source: AsyncIterable<Uint8Array | Uint8Array[]> | Iterable<Uint8Array | Uint8Array[]>,
  ): AsyncIterable<Constant> {
    for await (const pkt of source) {
      if (globalThis.Array.isArray(pkt)) {
        for (const p of (pkt as any)) {
          yield* [Constant.decode(p)];
        }
      } else {
        yield* [Constant.decode(pkt as any)];
      }
    }
  },

  fromJSON(object: any): Constant {
    return { interval: isSet(object.interval) ? globalThis.Number(object.interval) : 0 };
  },

  toJSON(message: Constant): unknown {
    const obj: any = {};
    if (message.interval !== 0) {
      obj.interval = Math.round(message.interval);
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Constant>, I>>(base?: I): Constant {
    return Constant.fromPartial(base ?? ({} as any));
  },
  fromPartial<I extends Exact<DeepPartial<Constant>, I>>(object: I): Constant {
    const message = createBaseConstant();
    message.interval = object.interval ?? 0;
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
