module github.com/aperturerobotics/util

go 1.19

replace (
	github.com/sirupsen/logrus => github.com/aperturerobotics/logrus v1.9.1-0.20221224130652-ff61cbb763af // aperture
	google.golang.org/protobuf => github.com/aperturerobotics/protobuf-go v1.30.1-0.20230428014030-7089409cbc63 // aperture
)

require (
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/fsnotify/fsnotify v1.6.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.9.2
	google.golang.org/protobuf v1.30.0
)

require golang.org/x/sys v0.0.0-20220908164124-27713097b956 // indirect
