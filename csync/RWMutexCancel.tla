------------------------- MODULE RWMutexCancel -------------------------
\* Models cancellation of the last waiting writer while an existing reader
\* remains active and another reader is queued. Other acquisition/release
\* interleavings are outside this focused notification contract.
EXTENDS TLC
WakeOnCancel == TRUE
VARIABLES waiting, queued, notified, acquired
vars == <<waiting, queued, notified, acquired>>
Init == /\ waiting = TRUE /\ queued = TRUE
        /\ notified = FALSE /\ acquired = FALSE
CancelWriter == /\ waiting /\ waiting' = FALSE
                /\ notified' = WakeOnCancel
                /\ UNCHANGED <<queued, acquired>>
AcquireReader == /\ ~waiting /\ queued /\ notified
                 /\ queued' = FALSE /\ notified' = FALSE
                 /\ acquired' = TRUE /\ UNCHANGED waiting
Done == /\ acquired /\ UNCHANGED vars
Next == CancelWriter \/ AcquireReader \/ Done
Spec == Init /\ [][Next]_vars /\ WF_vars(AcquireReader)
ReaderNotStranded == (queued /\ ~waiting) => notified
Progress == ~waiting ~> acquired
=============================================================================
