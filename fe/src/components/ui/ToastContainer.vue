<script setup lang="ts">
import { useUiStore } from '@/stores/ui';

const uiStore = useUiStore();

function getToastClass(type: string): string {
  switch (type) {
    case 'success':
      return 'bg-success text-success-foreground';
    case 'error':
      return 'bg-destructive text-destructive-foreground';
    case 'warning':
      return 'bg-warning text-warning-foreground';
    default:
      return 'bg-card text-card-foreground border';
  }
}

function getToastIcon(type: string): string {
  switch (type) {
    case 'success':
      return '✓';
    case 'error':
      return '✕';
    case 'warning':
      return '⚠';
    default:
      return 'ℹ';
  }
}
</script>

<template>
  <div class="fixed bottom-4 right-4 z-50 flex flex-col gap-2">
    <TransitionGroup name="toast">
      <div
        v-for="toast in uiStore.toasts"
        :key="toast.id"
        :class="[
          'px-4 py-3 rounded-lg shadow-lg flex items-center gap-3 min-w-[300px] max-w-[400px]',
          getToastClass(toast.type)
        ]"
      >
        <span class="text-lg font-bold">{{ getToastIcon(toast.type) }}</span>
        <span class="flex-1 text-sm">{{ toast.message }}</span>
        <button
          @click="uiStore.removeToast(toast.id)"
          class="opacity-70 hover:opacity-100 transition-opacity"
        >
          ✕
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s ease;
}

.toast-enter-from {
  opacity: 0;
  transform: translateX(100%);
}

.toast-leave-to {
  opacity: 0;
  transform: translateX(100%);
}
</style>
