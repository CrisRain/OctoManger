import { defineStore } from "pinia";
import { ref } from "vue";

export const useCommandPaletteStore = defineStore("command-palette", () => {
  const isOpen = ref(false);
  const query = ref("");

  function open() {
    isOpen.value = true;
    query.value = "";
  }

  function close() {
    isOpen.value = false;
  }

  return {
    isOpen,
    query,
    open,
    close,
  };
});
