from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class StringFilter(_message.Message):
    __slots__ = ("empty", "not_empty", "value", "values", "re", "has_prefix", "has_suffix", "contains")
    EMPTY_FIELD_NUMBER: _ClassVar[int]
    NOT_EMPTY_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    VALUES_FIELD_NUMBER: _ClassVar[int]
    RE_FIELD_NUMBER: _ClassVar[int]
    HAS_PREFIX_FIELD_NUMBER: _ClassVar[int]
    HAS_SUFFIX_FIELD_NUMBER: _ClassVar[int]
    CONTAINS_FIELD_NUMBER: _ClassVar[int]
    empty: bool
    not_empty: bool
    value: str
    values: _containers.RepeatedScalarFieldContainer[str]
    re: str
    has_prefix: str
    has_suffix: str
    contains: str
    def __init__(self, empty: _Optional[bool] = ..., not_empty: _Optional[bool] = ..., value: _Optional[str] = ..., values: _Optional[_Iterable[str]] = ..., re: _Optional[str] = ..., has_prefix: _Optional[str] = ..., has_suffix: _Optional[str] = ..., contains: _Optional[str] = ...) -> None: ...
