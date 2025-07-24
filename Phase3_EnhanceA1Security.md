/analysis Phase III: A1 Security  

TASKS:
1. In `pkg/a1/auth.go`, remove any bypass; implement `jwt.ParseWithClaims` + RSA public-key validation.
2. Create `pkg/a1/authz.go` with role checks: admin, operator, viewer.
3. Add integration tests in `pkg/a1/auth_test.go` validating valid/invalid tokens and RBAC enforcement.

VALIDATION:
!git diff pkg/a1/auth.go pkg/a1/authz.go
!go test pkg/a1 -coverprofile=coverage_a1.out
!go tool cover -func=coverage_a1.out | tail -1
