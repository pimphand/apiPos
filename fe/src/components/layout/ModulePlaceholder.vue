<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { useUiStore } from '@/stores/ui';
import { navigateTo } from '@/lib/navigation';
import { can, type Permission } from '@/lib/rbac';

interface Props {
  title: string;
  description?: string;
  icon?: string;
  permission?: Permission;
  comingFrom?: string;
  hint?: string;
}

const props = withDefaults(defineProps<Props>(), {
  description: '',
  icon: '🚧',
  hint: 'This module is part of the upcoming release and will be implemented in a later phase.',
});

const authStore = useAuthStore();
const uiStore = useUiStore();

const ready = ref(false);
const allowed = ref(true);

onMounted(() => {
  uiStore.loadTheme();
  if (!authStore.isAuthenticated) {
    navigateTo('/login');
    return;
  }
  if (props.permission) {
    allowed.value = can(authStore.currentUser, props.permission);
  }
  ready.value = true;
});
</script>

<template>
  <div v-if="ready" class="flex flex-col gap-6">
    <!-- Page header -->
    <div class="flex items-start justify-between gap-4">
      <div>
        <h2 class="text-2xl font-bold">{{ title }}</h2>
        <p v-if="description" class="text-muted-foreground mt-1">
          {{ description }}
        </p>
      </div>

      <div v-if="allowed" class="flex items-center gap-2">
        <span class="text-xs text-muted-foreground">{{ comingFrom || '' }}</span>
      </div>
    </div>

    <!-- Permission denied -->
    <div
      v-if="!allowed"
      class="flex flex-col items-center justify-center py-24 text-center bg-card rounded-xl border"
    >
      <div class="flex items-center justify-center size-16 rounded-full bg-muted mb-4">
        <span class="text-3xl">🔒</span>
      </div>
      <h3 class="text-xl font-semibold">Access restricted</h3>
      <p class="text-muted-foreground mt-2 max-w-sm">
        You don't have permission to view {{ title.toLowerCase() }}. Contact an
        administrator if you believe this is a mistake.
      </p>
    </div>

    <!-- Empty state -->
    <div
      v-else
      class="flex flex-col items-center justify-center py-24 text-center bg-card rounded-xl border"
    >
      <div class="flex items-center justify-center size-16 rounded-full bg-muted mb-4">
        <span class="text-3xl">{{ icon }}</span>
      </div>
      <h3 class="text-xl font-semibold">{{ title }} coming soon</h3>
      <p class="text-muted-foreground mt-2 max-w-md">{{ hint }}</p>
      <a
        href="/dashboard"
        class="mt-6 inline-flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors"
      >
        ← Back to Dashboard
      </a>
    </div>
  </div>

  <div v-else class="flex items-center justify-center h-[60vh]">
    <div class="text-center">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
      <p class="mt-4 text-sm text-muted-foreground">Loading...</p>
    </div>
  </div>
</template>
