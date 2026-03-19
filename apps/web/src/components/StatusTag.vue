<script setup lang="ts">
import { computed } from "vue";
import { getStatusLabel, getStatusColor, getStatusDotClass } from "@/shared/utils/status";

const props = defineProps<{ status?: string }>();

const label = computed(() => getStatusLabel(props.status ?? ""));
const color = computed(() => getStatusColor(props.status ?? ""));
const dotClass = computed(() => getStatusDotClass(props.status ?? ""));
</script>

<template>
  <div class="status-cell status-chip">
    <span class="status-dot-large" :class="dotClass" />
    <ui-tag :color="color" class="status-tag-pill status-chip-label">{{ label }}</ui-tag>
  </div>
</template>
