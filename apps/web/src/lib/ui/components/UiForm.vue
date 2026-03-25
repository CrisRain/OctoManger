<script setup lang="ts">
import { useAttrs } from "vue";
import { cx } from "../utils";

const attrs = useAttrs();
</script>

<template>
  <div v-bind="{ ...attrs, class: undefined }" :class="cx('ui-form space-y-5', attrs.class as string)">
    <slot />
  </div>
</template>
