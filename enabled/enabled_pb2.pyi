from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from typing import ClassVar as _ClassVar

DESCRIPTOR: _descriptor.FileDescriptor

class Enabled(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DEFAULT: _ClassVar[Enabled]
    ENABLE: _ClassVar[Enabled]
    DISABLE: _ClassVar[Enabled]
DEFAULT: Enabled
ENABLE: Enabled
DISABLE: Enabled
