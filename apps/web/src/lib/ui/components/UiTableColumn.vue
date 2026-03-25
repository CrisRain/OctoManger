<script setup lang="ts">
// Renderless config component — consumed by UiTable via slot introspection
defineProps<{
  title?: string;
  dataIndex?: string;
  align?: "left" | "center" | "right";
}>();
</script>

<template><!-- renderless --></template>
