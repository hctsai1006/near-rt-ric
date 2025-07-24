/analysis Begin Phase I:  
- Remove `dashboard-master/dashboard-master/` and `xAPP_dashboard-master/`.  
- Consolidate config files in `/config`.  
- Update `Makefile` to reference `frontend-dashboard/` instead.  
- Verify with:
  !rm -rf dashboard-master/dashboard-master xAPP_dashboard-master
  !git add -A
  !git diff --cached --stat