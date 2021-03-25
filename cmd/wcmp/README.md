# wcmp

The main purpose of this example is to demonstrate how to use P4Runtime with
action profiles / action selector. The wcmp defines an action selector, and we
use the P4Runtime "one shot" method to insert match entries in the table
referencing this action selector.

By itself, the control plane program does not do anything useful. It simply
inserts sample entries into the "wcmp_group" match table. It would be pretty
straightforward to modify the control plane program to configure static routes,
e.g. in order to provide routing for a well-known Mininet topology.
