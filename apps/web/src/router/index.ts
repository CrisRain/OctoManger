import { createRouter, createWebHistory } from "vue-router";
import { routes } from "./registry";

export const router = createRouter({
  history: createWebHistory(),
  routes,
});
