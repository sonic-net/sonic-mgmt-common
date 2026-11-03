# Management schemas and configuration are data; preserve dh_fixperms's layout.
/type=file/ { sub(/mode=[0-7]+/, "mode=0644") }
