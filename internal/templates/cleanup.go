package templates

var CleanupNBDB = `export nbstatus=$(ovs-appctl -t /var/run/ovn/ovnnb_db.ctl cluster/status OVN_Northbound)
echo "current northbound status"
echo "$nbstatus"
echo "searching for node {{ .NodeAddress }}"
nodeID=$(ovs-appctl -t /var/run/ovn/ovnnb_db.ctl cluster/status OVN_Northbound | grep '{{ .NodeAddress }}' | awk '{print $1}')
if [ -n "$nodeID" ]
then
  echo "removing node with id $nodeID"
  ovs-appctl -t /var/run/ovn/ovnnb_db.ctl cluster/kick OVN_Northbound $nodeID
  echo "removed node id $nodeID with address {{ .NodeAddress }}"
  echo "current northbound status"
  ovs-appctl -t /var/run/ovn/ovnnb_db.ctl cluster/status OVN_Northbound
fi`

var CleanupSBDB = `export sbstatus=$(ovs-appctl -t /var/run/ovn/ovnsb_db.ctl cluster/status OVN_Southbound)
echo "current southbound status"
echo "$sbstatus"
nodeID=$(ovs-appctl -t /var/run/ovn/ovnsb_db.ctl cluster/status OVN_Southbound| grep '{{ .NodeAddress }}' | awk '{print $1}')
if [ -n "$nodeID" ]
then
  ovs-appctl -t /var/run/ovn/ovnsb_db.ctl cluster/kick OVN_Southbound $nodeID
  echo "removed node id $nodeID with address {{ .NodeAddress }}"
  echo "current southbound status"
  ovs-appctl -t /var/run/ovn/ovnsb_db.ctl cluster/status OVN_Southbound
fi`

var CleanupChassis = `chassis=$(ovn-sbctl --columns=name find  chassis hostname={{ .Hostname }} | awk -F ":" '{print $2}' | tr -d '"')
if [ -n "$chassis" ]
then
  ovn-sbctl chassis-del $chassis
fi
ovn-sbctl show`
