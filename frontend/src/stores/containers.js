import { defineStore, acceptHMRUpdate } from 'pinia'

function byteStr(b) {
  if (b < 1024) {
    return `${b}B`
  }
  let suf = 'B'
  if (b < 1048576) {
    suf = 'KB'
    b = b / 1024
  } else if (b < 1073741824) {
    suf = 'MB'
    b = b / 1048576
  } else {
    suf = 'GB'
    b = b / 1073741824
  }
  return `${b.toFixed(1)}${suf}`
}
const columns = [
  {
    name: 'name',
    required: true,
    label: 'Name',
    align: 'left',
    field: (row) => row.Name,
    format: (val) => `${val}`,
    sortable: true,
  },
  { name: 'url', align: 'center', label: 'URL', field: (row) => row.URL, sortable: true },
  { name: 'image', label: 'Image', field: (row) => row.Image, sortable: true },
  {
    name: 'active',
    label: 'Active',
    field: (row) => row.Active,
    format: (val) => (val ? 'active' : 'inactive'),
  },
  {
    name: 'cpu',
    label: 'CPU %',
    field: (row) => row.CPUUsage,
    sortable: true,
    format: (val) => val.toFixed(1),
  },
  {
    name: 'mem_usage',
    label: 'Mem Usage (%)',
    field: (row) => row.MemUsage,
    sortable: true,
    format: (val) => val.toFixed(2),
  },
  {
    name: 'used_memory',
    label: 'Used memory',
    field: (row) => row.UsedMemory,
    sortable: true,
    format: (val) => byteStr(val),
  },
  {
    name: 'avail_memory',
    label: 'Available memory',
    field: (row) => row.AvailableMemory,
    sortable: true,
    format: (val) => byteStr(val),
  },
  {
    name: 'network',
    label: 'Network I/O',
    field: (row) => row.Net,
    format: (n) => `${byteStr(n.i)} / ${byteStr(n.o)}`,
  },
  { name: 'rps', label: 'RPS', field: (row) => row.rate_limit },
]

export const useContainerStore = defineStore('container', {
  state: () => ({
    containers: [],
  }),

  getters: {
    columns: () => columns,
    rows: (state) =>
      state.containers.map(function (c) {
        return {
          Name: c.Name,
          URL: `//${c.Subdomain}.${location.host}`,
          Image: c.Image,
          Active: c.Active,
          CPUUsage: c.Stats.CPUUsage,
          MemUsage: c.Stats.MemoryUsage,
          UsedMemory: c.Stats.UsedMemory,
          AvailableMemory: c.Stats.AvailableMemory,
          Net: { i: c.Stats.NetInput, o: c.Stats.NetOutput },
          rate_limit: c.rate_limit,
        }
      }),
  },

  actions: {
    async update() {
      try {
        const response = await fetch('/api/containers')
        if (!response.ok) {
          throw new Error(`Response status: ${response.status}`)
        }

        const json = await response.json()
        this.containers = json
        console.info('Containers', this.containers)
      } catch (error) {
        console.error(error.message)
      }
    },
    set(containers) {
      console.info('Setting containers', containers)
      this.containers = containers
    },
  },
})

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useContainerStore, import.meta.hot))
}
