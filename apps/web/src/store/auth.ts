import { defineStore } from "pinia";
import { ref } from "vue";
import { getAdminKey, setAdminKey } from "@/lib/auth";

export const useAuthStore = defineStore("auth", () => {
  const adminKey = ref(getAdminKey());

  function setKey(key: string) {
    adminKey.value = key;
    setAdminKey(key);
  }

  return { adminKey, setKey };
});
