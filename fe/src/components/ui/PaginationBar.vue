<script setup lang="ts">
import { computed } from 'vue';
import Button from '@/components/ui/button/Button.vue';

interface Props {
  page: number;
  perPage: number;
  total: number;
  maxVisible?: number;
}

const props = withDefaults(defineProps<Props>(), {
  maxVisible: 5,
});

const emit = defineEmits<{
  (e: 'update:page', value: number): void;
}>();

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.perPage)));

const pages = computed(() => {
  const current = Math.min(props.page, totalPages.value);
  const half = Math.floor(props.maxVisible / 2);
  let start = Math.max(1, current - half);
  let end = start + props.maxVisible - 1;
  if (end > totalPages.value) {
    end = totalPages.value;
    start = Math.max(1, end - props.maxVisible + 1);
  }
  const arr: (number | '...')[] = [];
  if (start > 1) {
    arr.push(1);
    if (start > 2) arr.push('...');
  }
  for (let i = start; i <= end; i++) arr.push(i);
  if (end < totalPages.value) {
    if (end < totalPages.value - 1) arr.push('...');
    arr.push(totalPages.value);
  }
  return arr;
});

function go(p: number) {
  const target = Math.min(Math.max(1, p), totalPages.value);
  if (target !== props.page) emit('update:page', target);
}
</script>

<template>
  <nav v-if="totalPages > 1" class="flex items-center gap-1" aria-label="Pagination">
    <Button
      variant="outline"
      size="icon"
      :disabled="page <= 1"
      aria-label="Previous page"
      @click="go(page - 1)"
    >
      ‹
    </Button>
    <template v-for="(p, i) in pages" :key="i">
      <span v-if="p === '...'" class="px-1 text-muted-foreground">…</span>
      <Button
        v-else
        :variant="p === page ? 'default' : 'outline'"
        size="icon"
        :aria-current="p === page ? 'page' : undefined"
        @click="go(p as number)"
      >
        {{ p }}
      </Button>
    </template>
    <Button
      variant="outline"
      size="icon"
      :disabled="page >= totalPages"
      aria-label="Next page"
      @click="go(page + 1)"
    >
      ›
    </Button>
  </nav>
</template>
