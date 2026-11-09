These files are captures of real BMCs, taken from the Redfish mockups in
https://github.com/cholcombe973/libredfish (tests/mockups, MIT licensed), which were
recorded with the DMTF Redfish-Mockup-Creator:

- dell_r750.json: Dell PowerEdge R750, iDRAC 9
- hpe_ilo5.json: HPE ProLiant DL385 Gen10 Plus v2, iLO 5
- lenovo_xcc.json: Lenovo ThinkSystem SR670 V2, XCC
- supermicro_hgx.json: Supermicro SYS-821GE-TNHR (HGX H100 8-GPU)
- nvidia_gb200.json: NVIDIA GB200 NVL compute tray

Each file maps a Redfish path to its response. The responses have been trimmed to the
properties the exporter reads during discovery and firmware collection.
