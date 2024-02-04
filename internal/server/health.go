package server

import (
	"api/config"
	"math"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type health struct {
	Name   string  `json:"name"`
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
}

func getServiceHealth() (health, error) {
	var health health

	config := config.Get()
	health.Name = config.Name

	cpuUsage, err := cpu.Percent(0, false)
	if err != nil {
		return health, err
	}
	health.CPU = cpuUsage[0]

	vm, err := mem.VirtualMemory()
	if err != nil {
		return health, err
	}
	health.Memory = float64(vm.Used) / math.Pow(1024, 2) // convert to MB

	return health, nil
}
