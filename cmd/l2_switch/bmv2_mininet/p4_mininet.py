# Copyright 2013-present Barefoot Networks, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

from mininet.net import Mininet
from mininet.node import Switch, Host
from mininet.log import setLogLevel, info


class P4Host(Host):
    def config(self, **params):
        r = super(Host, self).config(**params)

        self.defaultIntf().rename("eth0")

        for off in ["rx", "tx", "sg"]:
            cmd = "/sbin/ethtool --offload eth0 {} off".format(off)
            self.cmd(cmd)

        # disable IPv6
        self.cmd("sysctl -w net.ipv6.conf.all.disable_ipv6=1")
        self.cmd("sysctl -w net.ipv6.conf.default.disable_ipv6=1")
        self.cmd("sysctl -w net.ipv6.conf.lo.disable_ipv6=1")

        return r

    def describe(self):
        print("**********")
        print(self.name)
        print("default interface: {}\t{}\t{}".format(
            self.defaultIntf().name,
            self.defaultIntf().IP(),
            self.defaultIntf().MAC()
        ))
        print("**********")


class P4Switch(Switch):
    """P4 virtual switch"""
    device_id = 0

    def __init__(self, name, sw_path=None,
                 pcap_dump=False,
                 verbose=False,
                 device_id=None,
                 cpu_port=None,
                 **kwargs):
        Switch.__init__(self, name, **kwargs)
        assert (sw_path)
        self.sw_path = sw_path
        self.verbose = verbose
        logfile = '/tmp/p4s.{}.log'.format(self.name)
        self.output = open(logfile, 'w')
        self.pcap_dump = pcap_dump
        self.cpu_port = cpu_port
        if device_id is not None:
            self.device_id = device_id
            P4Switch.device_id = max(P4Switch.device_id, device_id)
        else:
            self.device_id = P4Switch.device_id
            P4Switch.device_id += 1

    @classmethod
    def setup(cls):
        pass

    def start(self, controllers):
        "Start up a new P4 switch"
        print("Starting P4 switch", self.name)
        args = [self.sw_path]
        for port, intf in self.intfs.items():
            if not intf.IP():
                self.cmd("sysctl -w net.ipv6.conf.{}.disable_ipv6=1".format(intf.name))
        for port, intf in self.intfs.items():
            if not intf.IP():
                args.extend(['-i', str(port) + "@" + intf.name])
        if self.cpu_port:
            args.extend(['-i', "64@" + self.cpu_port])
        if self.pcap_dump:
            args.append("--pcap")
        args.append("--no-p4")
        args.append("--log-console")
        args.extend(['--device-id', str(self.device_id)])
        P4Switch.device_id += 1
        logfile = '/tmp/p4s.{}.log'.format(self.name)
        print(' '.join(args))

        self.cmd(' '.join(args) + ' >' + logfile + ' 2>&1 &')

        print("switch has been started")

    def stop(self):
        "Terminate P4 switch."
        self.output.flush()
        self.cmd('kill %' + self.sw_path)
        self.cmd('wait')
        self.deleteIntfs()

    def attach(self, intf):
        "Connect a data port"
        assert (0)

    def detach(self, intf):
        "Disconnect a data port"
        assert (0)
