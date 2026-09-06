<script setup lang="ts">
import { computed } from 'vue';
import { useLocaleStore } from '@/stores/locale';
import { ORDER_CHANNEL_LABELS, type OrderChannel } from '@/types';
import { cn } from '@/lib/utils';

const props = defineProps<{
  modelValue: OrderChannel;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: OrderChannel];
}>();

const t = useLocaleStore().t;

const dineInActive = computed(() => props.modelValue === 'dinein');
const takeAwayActive = computed(() => props.modelValue === 'takeaway');

function selectDineIn() {
  emit('update:modelValue', 'dinein');
}
function selectTakeAway() {
  emit('update:modelValue', 'takeaway');
}
</script>

<template>
  <div class="flex items-center gap-2">
    <span class="text-sm text-muted-foreground shrink-0">{{ t('orderChannel.title') }}</span>
    <div class="inline-flex rounded-md border bg-background shadow-xs">
      <button
        :class="cn(
          'relative inline-flex items-center justify-center text-sm font-medium rounded-md h-9 px-3 transition-colors',
          dineInActive
            ? 'bg-primary text-primary-foreground shadow-sm'
            : 'text-muted-foreground hover:bg-muted',
        )"
        type="button"
        @click="selectDineIn()"
      >
        {{ t('orderChannel.dineIn') }}
        <span
          v-if="dineInActive"
          class="absolute inset-y-0 right-0 flex items-center pr-1"
        >
          <svg class="h-4 w-4 text-primary-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="4">
            <path d="M9 12l2 2 4-4" />
          </svg>
        </span>
      </button>
      <button
        :class="cn(
          'relative inline-flex items-center justify-center text-sm font-medium rounded-md h-9 px-3 transition-colors',
          takeAwayActive
            ? 'bg-primary text-primary-foreground shadow-sm'
            : 'text-muted-foreground hover:bg-muted',
        )"
        type="button"
        @click="selectTakeAway()"
      >
        {{ t('orderChannel.takeAway') }}
        <span
          v-if="takeAwayActive"
          class="absolute inset-y-0 right-0 flex items-center pr-1"
        >
          <svg class="h-4 w-4 text-primary-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="4">
            <path d="M9 12l2 2 4-4" />
          </svg>
        </span>
      </button>
    </div>
  </div>
</template>
