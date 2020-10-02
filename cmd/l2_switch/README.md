# l2_switch

This example includes a simple P4 program for a L2 learning switch, along with
its corresponding control plane program. The example does *not* implement the
Spanning Tree Protocol and as such you should not use it in topologies which
include loops.

If you are using bmv2 simple_switch_grpc, all you need to do to run the control
plane program is:

```bash
# from the root of the repo
make
./bin/l2_switch --verbose --ports <comma-separated list of switch port numbers>
```

If you want to give it a try, I suggest creating a Mininet topology with a
single simple_switch_grpc instance connecting 2 hosts:

```bash
# from the root of the repo
cd cmd/l2_switch/bmv2_mininet/
sudo python 1sw_demo.py
```

After that you can run `./bin/l2_switch`:

```bash
./bin/l2_switch --verbose
```

and use the `pingall` command in the Mininet CLI to check connectivity.
