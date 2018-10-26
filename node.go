package model

import (
	"encoding/json"
	"errors"
)

type NodeHook struct {
	Provision Hook
	Destroy   Hook
}

func (r NodeHook) MarshalJSON() ([]byte, error) {
	t := struct {
		Provision *Hook `json:",omitempty"`
		Destroy   *Hook `json:",omitempty"`
	}{}
	if r.Provision.HasTasks() {
		t.Provision = &r.Provision
	}
	if r.Destroy.HasTasks() {
		t.Destroy = &r.Destroy
	}
	return json.Marshal(t)
}

func (r NodeHook) HasTasks() bool {
	return r.Provision.HasTasks() ||
		r.Destroy.HasTasks()
}

// NodeSet contains the whole specification of a nodes et to create on a specific
// cloud provider
type NodeSet struct {
	// The name of the machines
	Name string
	// The number of machines to create
	Instances int
	// The ref to the provider where to create the machines
	Provider ProviderRef
	// The parameters related to the orchestrator used to manage the machines
	Orchestrator OrchestratorRef
	// Volumes attached to each node
	Volumes []Volume
	// Hooks for executing tasks around provisioning and destruction
	Hooks NodeHook
	// The labels associated with the nodeset
	Labels Labels
}

func (n NodeSet) DescType() string {
	return "NodeSet"
}

func (n NodeSet) DescName() string {
	return n.Name
}

func (r NodeSet) MarshalJSON() ([]byte, error) {
	t := struct {
		Name         string       `json:",omitempty"`
		Instances    int          `json:",omitempty"`
		Provider     Provider     `json:",omitempty"`
		Orchestrator Orchestrator `json:",omitempty"`
		Volumes      []Volume
		Hooks        *NodeHook `json:",omitempty"`
	}{
		Name:         r.Name,
		Instances:    r.Instances,
		Provider:     r.Provider.Resolve(),
		Orchestrator: r.Orchestrator.Resolve(),
		Volumes:      r.Volumes,
	}
	if r.Hooks.HasTasks() {
		t.Hooks = &r.Hooks
	}
	return json.Marshal(t)
}

func (r NodeSet) validate() ValidationErrors {
	vErrs := ValidationErrors{}
	vErrs.merge(r.Provider.validate())
	vErrs.merge(r.Orchestrator.validate())
	vErrs.merge(r.Hooks.Provision.validate())
	vErrs.merge(r.Hooks.Destroy.validate())
	return vErrs
}

func (r *NodeSet) merge(other NodeSet) {
	if r.Name != other.Name {
		panic(errors.New("cannot merge unrelated node sets (" + r.Name + " != " + other.Name + ")"))
	}
	if r.Instances < other.Instances {
		r.Instances = other.Instances
	}
	r.Provider.merge(other.Provider)
	r.Orchestrator.merge(other.Orchestrator)
	for id, v := range other.Volumes {
		r.Volumes[id].merge(v)
	}
	r.Hooks.Provision.merge(other.Hooks.Provision)
	r.Hooks.Destroy.merge(other.Hooks.Destroy)
	r.Labels = r.Labels.inherits(other.Labels)
}

func createNodeSets(env *Environment, yamlEnv *yamlEnvironment) map[string]NodeSet {
	res := map[string]NodeSet{}
	for name, yamlNodeSet := range yamlEnv.Nodes {
		if yamlNodeSet.Instances <= 0 {
			env.errors.addError(errors.New("instances must be a positive number"), env.location.appendPath("nodes."+name+".instances"))
		}
		res[name] = NodeSet{
			Name:         name,
			Instances:    yamlNodeSet.Instances,
			Provider:     createProviderRef(env, env.location.appendPath("nodes."+name+".provider.name"), yamlNodeSet.Provider),
			Orchestrator: createOrchestratorRef(env, env.location.appendPath("nodes."+name+".orchestrator"), yamlNodeSet.Orchestrator),
			Volumes:      createVolumes(env, env.location.appendPath("nodes."+name+".volumes"), yamlNodeSet.Volumes),
			Hooks: NodeHook{
				Provision: createHook(env, env.location.appendPath("nodes."+name+".hooks.provision"), yamlNodeSet.Hooks.Provision),
				Destroy:   createHook(env, env.location.appendPath("nodes."+name+".hooks.destroy"), yamlNodeSet.Hooks.Destroy),
			},
			Labels: yamlNodeSet.Labels,
		}
	}
	return res
}
