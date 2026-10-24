from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BackoffKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BackoffKind_UNKNOWN: _ClassVar[BackoffKind]
    BackoffKind_EXPONENTIAL: _ClassVar[BackoffKind]
    BackoffKind_CONSTANT: _ClassVar[BackoffKind]
BackoffKind_UNKNOWN: BackoffKind
BackoffKind_EXPONENTIAL: BackoffKind
BackoffKind_CONSTANT: BackoffKind

class Backoff(_message.Message):
    __slots__ = ("backoff_kind", "exponential", "constant")
    BACKOFF_KIND_FIELD_NUMBER: _ClassVar[int]
    EXPONENTIAL_FIELD_NUMBER: _ClassVar[int]
    CONSTANT_FIELD_NUMBER: _ClassVar[int]
    backoff_kind: BackoffKind
    exponential: Exponential
    constant: Constant
    def __init__(self, backoff_kind: _Optional[_Union[BackoffKind, str]] = ..., exponential: _Optional[_Union[Exponential, _Mapping]] = ..., constant: _Optional[_Union[Constant, _Mapping]] = ...) -> None: ...

class Exponential(_message.Message):
    __slots__ = ("initial_interval", "multiplier", "max_interval", "randomization_factor", "max_elapsed_time")
    INITIAL_INTERVAL_FIELD_NUMBER: _ClassVar[int]
    MULTIPLIER_FIELD_NUMBER: _ClassVar[int]
    MAX_INTERVAL_FIELD_NUMBER: _ClassVar[int]
    RANDOMIZATION_FACTOR_FIELD_NUMBER: _ClassVar[int]
    MAX_ELAPSED_TIME_FIELD_NUMBER: _ClassVar[int]
    initial_interval: int
    multiplier: float
    max_interval: int
    randomization_factor: float
    max_elapsed_time: int
    def __init__(self, initial_interval: _Optional[int] = ..., multiplier: _Optional[float] = ..., max_interval: _Optional[int] = ..., randomization_factor: _Optional[float] = ..., max_elapsed_time: _Optional[int] = ...) -> None: ...

class Constant(_message.Message):
    __slots__ = ("interval",)
    INTERVAL_FIELD_NUMBER: _ClassVar[int]
    interval: int
    def __init__(self, interval: _Optional[int] = ...) -> None: ...
