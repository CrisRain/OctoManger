<script setup lang="ts">
// Renderless config component — consumed by UiTable via slot introspection
defineProps<{
  title?: string;
  dataIndex?: string;
  width?: number | string;
  align?: "left" | "center" | "right";
}>();
</script>

<template><!-- renderless --></template>
