<template>
  <div v-show="show" :title="item.name">
    <!--title instead of tooltip above as the tooltip needs too much work with the current methods, i.e. not being loaded or staying open for "too long"-->
    <div class="item" :class="[item.isFile ? 'filewrapper gt-p-1 gt-ac' : '', store.selectedItem === '#diff-' + item.file?.NameHash ? 'selected' : '']">
      <!-- Files -->
      <SvgIcon
        v-if="item.isFile"
        :name="getIconForDiffType(item.file.Type)"
        :class="['svg-icon', getIconForDiffType(item.file.Type), 'status']"
      />
      <a
        v-if="item.isFile"
        class="file gt-ellipsis"
        :href="item.isFile ? '#diff-' + item.file.NameHash : ''"
      >{{ item.name }}</a>
      <!-- Directories -->
      <div v-if="!item.isFile" class="directory gt-p-1 gt-ac" @click.stop="handleClick(item.isFile)">
        <SvgIcon
          class="svg-icon"
          :name="collapsed ? 'octicon-chevron-right' : 'octicon-chevron-down'"
        />
        <SvgIcon
          class="svg-icon directory"
          name="octicon-file-directory-fill"
        />
        <span class="gt-ellipsis">{{ item.name }}</span>
      </div>
      <div v-show="!collapsed">
        <DiffFileTreeItem v-for="childItem in item.children" :key="childItem.name" :item="childItem" class="list"/>
      </div>
    </div>
  </div>
</template>

<script>
import {SvgIcon} from '../svg.js';
import {DiffTreeStore} from '../modules/stores.js';

export default {
  components: {SvgIcon},
  props: {
    item: {
      type: Object,
      required: true
    },
    show: {
      type: Boolean,
      required: false,
      default: true
    },
  },
  data: () => ({
    store: DiffTreeStore,
    collapsed: false,
  }),
  methods: {
    handleClick(itemIsFile) {
      if (itemIsFile) {
        return;
      }
      this.collapsed = !this.collapsed;
    },
    getIconForDiffType(pType) {
      const diffTypes = {
        0: 'sc-file-diff',
        1: 'sc-file-plus',
        2: 'sc-file-diff',
        3: 'sc-file-minus',
        4: 'sc-file-diff',
        5: 'sc-file-diff', // there is no octicon for copied, so modified should be ok
      };
      return diffTypes[pType];
    },
  },
};
</script>

<style scoped>
.svg-icon.status {
  float: right;
}

.svg-icon.file {
  color: var(--color-secondary-dark-7);
}

.svg-icon.directory {
  color: var(--color-primary);
}

.svg-icon.sc-file-diff {
  color: var(--sc-color-yellow-50);
}

.svg-icon.sc-file-plus {
  color: var(--sc-color-green-50);
}

.svg-icon.sc-file-minus {
  color: var(--sc-color-red-50);
}

.svg-icon.sc-file-diff {
  color: var(--color-teal);
}

/* .item {
  height: 36px;
  padding: 8px 10px;
  display: flex;
  align-items: center;
} */

.item.filewrapper {
  display: grid !important;
  grid-template-columns: 20px 7fr 1fr;
  padding-left: 18px !important;
}

.item.filewrapper:hover, div.directory:hover {
  color: var(--color-text);
  background: var(--color-hover);
  border-radius: 4px;
}

.item.filewrapper.selected {
  color: var(--color-text);
  background: var(--color-active);
  border-radius: 4px;
}

div.directory {
  display: grid;
  grid-template-columns: 18px 20px auto;
  user-select: none;
  cursor: pointer;
}

div.list {
  padding-bottom: 0 !important;
  padding-top: inherit !important;
}

a {
  text-decoration: none;
  color: var(--color-text);
}

a:hover {
  text-decoration: none;
  color: var(--color-text);
}
</style>
