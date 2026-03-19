<script setup lang="ts">
interface Props {
  value?: unknown;
  disabled?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  value: undefined,
  disabled: false,
});
</script>

<template>
  <option :value="(props.value as never)" :disabled="disabled">
    <slot />
  </option>
</template>
