/analysis Phase VI: Tests  

TASKS:
1. Add table-driven unit tests for pkg/e2, pkg/a1, pkg/o1 covering edge cases.
2. Create integration tests that simulate SCTP handshake, JWT flows, NETCONF RPCs.
3. Generate combined `coverage.out` and enforce threshold.

VALIDATION:
!git diff test/
!make test-coverage
