import { createRouter, createWebHistory } from "vue-router";
import { getAdminKey } from "@/lib/auth";
import { routes } from "./registry";
import { evaluateNavigationGuard } from "./guard";

export const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach(async (to) => {
  return evaluateNavigationGuard(to.path, to.fullPath, getAdminKey());
});
