# Exponential Backoff

This is a fork of [cenkalti/backoff](https://github.com/cenkalti/backoff) v4.0, with some unused features removed.

This package is a Go port of the exponential backoff algorithm from [Google's HTTP Client Library for Java][google-http-java-client].

[Exponential backoff][exponential backoff wiki]
is an algorithm that uses feedback to multiplicatively decrease the rate of some process,
in order to gradually find an acceptable rate.
The retries exponentially increase and stop increasing when a certain threshold is met.
