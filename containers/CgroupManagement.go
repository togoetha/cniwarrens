package containers

import (
	"fmt"

	cgroups "github.com/containerd/cgroups"
	cgroupsv2 "github.com/containerd/cgroups/v2"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

var MainGroup *cgroups.Cgroup
var Manager *cgroupsv2.Manager
var Cgroups map[string]*cgroups.Cgroup

func InitCgroups() {

	if cgroups.Mode() == cgroups.Unified {
		res := cgroupsv2.Resources{}
		m, err := cgroupsv2.NewSystemd("/", "fledge.slice", -1, &res)
		if err != nil {
			fmt.Println(err)
		}
		Manager = m
	} else {
		maingroup, err := cgroups.New(cgroups.V1, cgroups.StaticPath("/vkubelet"), &specs.LinuxResources{})
		if err != nil {
			fmt.Println(err)
		}

		MainGroup = &maingroup
		//fmt.Println(MainGroup)
		Cgroups = make(map[string]*cgroups.Cgroup)
	}
}

func GetCgroup(namespace string, podname string, container string) string {
	cgName := fmt.Sprintf("%s-%s-%s", namespace, podname, container)
	return cgName
}

func CreateCgroupIfNotExists(namespace string, podname string, container string) string {
	cgName := GetCgroup(namespace, podname, container)
	if !CgroupExists(cgName) {
		CreateCgroup(cgName)
	}
	return cgName
}

func CreateCgroup(cgName string) {
	fmt.Println("CreateCgroup")
	//cmd := fmt.Sprintf("cgcreate -g memory,cpu:vkubelet/%s", cgName)
	//utils.ExecCmdBash(cmd)
	/*cmd := fmt.Sprintf("mkdir -p /sys/fs/cgroup/memory/%s", cgName)
	utils.ExecCmdBash(cmd)
	cmd = fmt.Sprintf("mkdir -p /sys/fs/cgroup/cpu/%s", cgName)
	utils.ExecCmdBash(cmd)*/
	newGroup, err := (*MainGroup).New(cgName, &specs.LinuxResources{})
	if err != nil {
		fmt.Println("Error creating new group")
	}
	Cgroups[cgName] = &newGroup
}

func CgroupExists(cgName string) bool {
	fmt.Println("CgroupExists")
	//val, exists := Cgroups[cgName]
	//return exists || val == nil

	//cmd := fmt.Sprintf("cgget -g memory:vkubelet/%s", cgName)
	//cmd := fmt.Sprintf("cat /sys/fs/cgroup/memory/%s/memory.limit_in_bytes", cgName)
	//utils.ExecCmdBash(cmd)
	//return err == nil
	return false
}

func DeleteCgroup(cgName string) {
	fmt.Println("DeleteCgroup")
	//cmd := fmt.Sprintf("cgdelete memory,cpu:vkubelet/%s", cgName)
	//cmd := fmt.Sprintf("rmdir /sys/fs/cgroup/memory/%s", cgName)
	//utils.ExecCmdBash(cmd)
	//cmd = fmt.Sprintf("rmdir /sys/fs/cgroup/cpu/%s", cgName)
	//utils.ExecCmdBash(cmd)
	group := Cgroups[cgName]
	(*group).Delete()
	Cgroups[cgName] = nil
}

func SetMemoryLimit(cgName string, limit int64) {
	fmt.Println("SetMemoryLimit")
	//cmd := fmt.Sprintf("echo %d > /sys/fs/cgroup/memory/%s/memory.limit_in_bytes", limit, cgName)
	//cmd := fmt.Sprintf("cgset -r memory.limit_in_bytes=%d vkubelet/%s", limit, cgName)
	//utils.ExecCmdBash(cmd)

	cgroup := *Cgroups[cgName]
	specs := &specs.LinuxResources{
		Memory: &specs.LinuxMemory{
			Limit: &limit,
		},
	}
	cgroup.Update(specs)
}

func SetCpuLimit(cgName string, cpus float64) {
	fmt.Println("SetCPULimit")
	//cpu.cfs_period_us=100000
	//cpu.cfs_quota=100000 * cpus?
	//cmd := fmt.Sprintf("echo %d > /sys/fs/cgroup/cpu/%s/cpu.cfs_period_us", 100000, cgName)
	//cmd := fmt.Sprintf("cgset -r cpu.cfs_period_us=%d vkubelet/%s", 100000, cgName)
	//utils.ExecCmdBash(cmd)
	//cmd = fmt.Sprintf("echo %d > /sys/fs/cgroup/cpu/%s/cpu.cfs_quota_us", int64(100000*cpus), cgName)
	//cmd = fmt.Sprintf("cgset -r cpu.cfs_quota_us=%d vkubelet/%s", int64(100000*cpus), cgName)
	//utils.ExecCmdBash(cmd)
	period := uint64(100000)
	quota := int64(100000 * cpus)

	cgroup := *Cgroups[cgName]
	specs := &specs.LinuxResources{
		CPU: &specs.LinuxCPU{
			Period: &period,
			Quota:  &quota,
		},
	}
	cgroup.Update(specs)
}

func MovePid(cgName string, pid uint64) {
	fmt.Println("MovePID")
	//cmd := fmt.Sprintf("cgclassify -g memory,cpu:vkubelet/%s %d", cgName, pid)
	//utils.ExecCmdBash(cmd)
	cgroup := *Cgroups[cgName]
	cgroup.AddProc(pid)
}
