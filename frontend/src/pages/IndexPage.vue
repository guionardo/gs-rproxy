<template>
  <q-page class="flex flex-center">
    <q-table title="Containers" :rows="rows" :columns="columns" row-key="name">
      <template v-slot:body-cell-url="props">
        <q-td :props="props">
          <a :href="props.row.URL">{{ props.row.URL }}</a>
        </q-td>
      </template>
    </q-table>
  </q-page>
</template>

<script setup>
import { useContainerStore } from 'stores/containers'
import { storeToRefs } from 'pinia'

const store = useContainerStore()
const { columns, rows } = storeToRefs(store)

const eventSource = new EventSource('/api/events')
eventSource.onmessage = function (event) {
  const cnt = JSON.parse(event.data)
  store.set(cnt)
}
</script>
