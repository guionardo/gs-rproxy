<template>
  <q-layout view="lHh Lpr lFf">
    <q-header elevated>
      <q-toolbar>
        <div><img src="header_logo.svg" height="48"></div>
        <q-toolbar-title> GS-RProxy </q-toolbar-title>
        <q-btn href="/metrics" target="new" title="Prometheus metrics" flat round dense>
          <q-icon name="query_stats" />
        </q-btn>
        <div>v0.1.0</div>
      </q-toolbar>
    </q-header>

    <q-page-container>
      <router-view />
    </q-page-container>
  </q-layout>
</template>

<script setup></script>
