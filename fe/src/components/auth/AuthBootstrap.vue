<script setup lang="ts">
import { onMounted } from 'vue';
import { useAuthStore } from '@/stores/auth';

const authStore = useAuthStore();

onMounted(() => {
  // Restore a persisted session (refresh token) on every protected page load.
  // Idempotent — no-op if a live session already exists. Pinia reactivity
  // updates the sidebar/user as soon as the session resolves.
  authStore.restoreSession();
});
</script>

<template>
  <div style="display: none" aria-hidden="true"></div>
</template>
