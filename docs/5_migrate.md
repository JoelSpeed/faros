---
layout: default
title: Migration to v0.7
permalink: /migrate
nav_order: 5
---

This document is only applicable if you are upgrading from a version before
v0.7.0 to a later version. You can safely ignore this document if this is
not the case for you.


## ClusterGitTrack migration plan

Kubernetes disallows having owner relationships that go from one namespace
into another or from a namespaced object to a cluster-scoped one. Previously,
Faros has been able to create these, and because of limitations in the
Kubernetes garbage collector, it hasn’t been immediately obvious that this
shouldn’t have worked.

This document sets out how to check whether you might be vulnerable to this
issue and how to mitigate it when upgrading to Faros version 0.7.0

## Checking if you’re impacted

If you have any ClusterGitTrackObjects in your cluster, you are impacted. You
can check this by running `kubectl get clustergittrackobjects`

If you don’t have any ClusterGitTrackObjects,
you might still be impacted. Run the tool
[here](https://github.com/pusher/faros/tree/master/hack/namespacecheck)
to check if you have any GitTrackObjects owned by GitTracks in a different
namespace.

If neither of these are applicable to your setup, you are not impacted. The
only change required is to add the `--gittrack-mode=Enabled` flag to your
Faros deployment when upgrading to version 0.7.0 or greater.

The rest of this document sets out how to migrate in the case that you
are impacted.

## Migrating ClusterGitTrackObjects

If you have ClusterGitTrackObjects in your setup, then you will have to
migrate those to being managed by ClusterGitTracks

1. Apply the ClusterGitTrack custom resource definition to your cluster. It'll
be needed for running later versions of Faros
2. Scale down your Faros deployments so that there are no active Faros pods
 running
3. Remove all `ownerReferences` from ClusterGitTrackObjects. For small setups, this can be done manually, but bigger setups can be done programmatically, for example, using `jq`:

	```
	# kubectl get -a clustergittrackobjects -o json > /tmp/cgtos.json
	# jq '.items[].metadata.ownerReferences = null' /tmp/cgtos.json > /tmp/newcgtos.json
	# kubectl apply -f /tmp/newcgtos.json
	```

	You can check that all ClusterGitTrackObjects are unowned with the following jq expression

	```
	# kubectl get clustergittrackobject -o json | jq '.items[].metadata | select(.ownerReferences == null) | .name' -r
	```

4. For every `GitTrack` which previously owned a `ClusterGitTrackObject`,
create a `ClusterGitTrack` that matches its target. If you used the
`namespacecheck` tool to check for `ClusterGitTracks`, it should have written
a file with all the required `ClusterGitTracks` to apply.
5. Create a new deployment of Faros with the flags `--gittrack-mode=Disabled`
and `--clustergittrack-mode=ExcludeNamespaced`. This should adopt all ClusterGitTrackObjects so they are owned by ClusterGitTracks
6. Check that ClusterGitTrackObjects are now owned by ClusterGitTracks. This can be done with the following jq expression

	```
	# kubectl get clustergittrackobject -o json | jq '.items[].metadata | select(.ownerReferences != null and .ownerReferences[].kind != "ClusterGitTrack") | .name' -r
	```

A Faros deployment must handle ClusterGitTracks. If you have one faros for the entire cluster, you can add the `--clustergittrack-mode=IncludeNamespaced` flag to it.[^1]

## Migrating GitTrackObjects

If the tool for checking GitTrackObjects didn’t find any cross-namespace
references, you are good to go, just add the `--gittrack-mode=Enabled`
to all your existing Faros deployments

If you did find objects owned across namespaces, you’ll have to take steps
to make sure that they are owned by a parent within their own namespace

1. Scale down your Faros deployments so that there are no active Faros pods
running
2. Remove all ownerReferences from GitTrackObjects. You can use the shell snippet from the ClusterGitTrack section to do this.

If you have a setup where all your GitTracks live in one namespace, follow
these steps

1. Create ClusterGitTracks matching each of your current GitTracks
2. Remove the existing GitTracks
3. Start a Faros with `--clustergittrack-mode=IncludeNamespaced` and
`--gittrack-mode=Disabled`

If you have a setup where GitTracks are distributed amongst multiple
namespaces, follow these steps

1. Create a new deployment of Faros with the flags `--gittrack-mode=Enabled`
and `--clustergittrack-mode=Disabled`, with no namespace (meaning it will
handle all namespaces)
2. For each GitTrack, inspect them for `status.ignoredFiles` saying `namespace
$NAMESPACE is not managed by this GitTrack`
3. Move your resources in git and update your GitTracks so that each GitTrack
manages a single namespace that the GitTrack lives in (you can have multiple
GitTrack per namespace, but only one namespace to a GitTrack)
4. Turn off the new deployment of Faros and scale up your old Faros deployments
with `--gittrack-mode=Enabled`

Once these steps are done, you can run the namespacecheck tool again to verify that there are no cross-namespace references.

[^1]: For Kubernetes internals reasons, a Faros controller cannot both handle a single namespace and `ClusterGitTracks`. If you only have namespaced Faros controllers, you will need to add a new deployment handling only `ClusterGitTracks`.
