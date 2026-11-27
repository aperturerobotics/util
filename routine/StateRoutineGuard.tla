------------------------- MODULE StateRoutineGuard -------------------------
EXTENDS Naturals, FiniteSets, TLC
\* RoutineContainer.transition owns state/template updates under bcast.
\* SetState and SwapValue use SetState below; SetStateRoutine changes fn;
\* SetContext/ClearContext use SetContext. Constructors reach every combination
\* of empty/nonempty state, callback and context before the first execution.
\* RestartRoutine repeats a context-enabled installation. GetState observes
\* the atomic state/template pair. WaitExited observes completion separately.
\* Three generations and three updates include replacement through a nil gap.
\* execute's predecessor receive, callback, and channel close can interleave.
\* Retry timing, errors, equality callbacks and exit notification delivery are
\* outside this guard model; unchanged state skips replacement in the code.
CONSTANT MaxRuns
VARIABLES state, fn, root, bound, issued, current, tail, phase, canceled,
          closed, predecessor, captured, updates, returned, expected
vars == <<state, fn, root, bound, issued, current, tail, phase, canceled,
          closed, predecessor, captured, updates, returned, expected>>
Enabled(s, f, c) == s # 0 /\ f /\ c
Init ==
  /\ state \in {0, 1}
  /\ fn \in BOOLEAN
  /\ root \in BOOLEAN
  /\ bound = IF fn THEN state ELSE 0
  /\ issued = IF Enabled(state, fn, root) THEN 1 ELSE 0
  /\ current = issued
  /\ tail = issued
  /\ phase = [i \in 1..MaxRuns |-> IF i = current THEN "waiting" ELSE "unused"]
  /\ canceled = {}
  /\ closed = {0}
  /\ predecessor = [i \in 1..MaxRuns |-> 0]
  /\ captured = [i \in 1..MaxRuns |-> IF i = current THEN state ELSE 0]
  /\ updates = 0
  /\ returned = 0
  /\ expected = 0
Transition(s, f, c) ==
  LET launch == Enabled(s, f, c)
      n == issued + 1
  IN
  /\ updates < 3
  /\ ~launch \/ issued < MaxRuns
  /\ state' = s
  /\ fn' = f
  /\ root' = c
  /\ bound' = IF f THEN s ELSE 0
  /\ canceled' = canceled \cup (IF current # 0 THEN {current} ELSE {})
  /\ returned' = tail
  /\ expected' = tail
  /\ issued' = IF launch THEN n ELSE issued
  /\ current' = IF launch THEN n ELSE 0
  /\ tail' = IF launch THEN n ELSE tail
  /\ phase' = IF launch THEN [phase EXCEPT ![n] = "waiting"] ELSE phase
  /\ predecessor' = IF launch THEN [predecessor EXCEPT ![n] = tail] ELSE predecessor
  /\ captured' = IF launch THEN [captured EXCEPT ![n] = s] ELSE captured
  /\ updates' = updates + 1
  /\ UNCHANGED closed
SetState == \E s \in {0, 1, 2} : s # state /\ Transition(s, fn, root)
SetStateRoutine == Transition(state, ~fn, root)
SetContext == Transition(state, fn, ~root)
RestartRoutine == root /\ bound # 0 /\ Transition(state, fn, root)
Invoke(i) ==
  /\ phase[i] = "waiting"
  /\ predecessor[i] \in closed
  /\ i \notin canceled
  /\ phase' = [phase EXCEPT ![i] = "running"]
  /\ UNCHANGED <<state, fn, root, bound, issued, current, tail, canceled,
                  closed, predecessor, captured, updates, returned, expected>>
Exit(i) ==
  /\ phase[i] = "running" \/ (phase[i] = "waiting" /\ i \in canceled)
  /\ predecessor[i] \in closed
  /\ phase' = [phase EXCEPT ![i] = "exited"]
  /\ closed' = closed \cup {i}
  /\ UNCHANGED <<state, fn, root, bound, issued, current, tail, canceled,
                  predecessor, captured, updates, returned, expected>>
Next == SetState \/ SetStateRoutine \/ SetContext \/ RestartRoutine
        \/ (\E i \in 1..MaxRuns : Invoke(i) \/ Exit(i))
Done == /\ updates = 3 /\ (1..issued) \subseteq closed /\ UNCHANGED vars
Spec == Init /\ [][Next \/ Done]_vars
AtomicState == bound = IF fn THEN state ELSE 0
ReplacedCanceled == \A i \in 1..issued : i # current => i \in canceled
ReplacementOrdering ==
  /\ Cardinality({i \in 1..issued : phase[i] = "running"}) <= 1
  /\ \A i \in 1..issued : phase[i] = "running" => (0..(i-1)) \subseteq closed
CompletionChain == \A i \in closed : (0..i) \subseteq closed
ReturnedFence == returned = expected
CapturedState == current # 0 => captured[current] = bound
=============================================================================
