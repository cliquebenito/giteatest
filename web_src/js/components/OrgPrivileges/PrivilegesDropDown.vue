<template>
  <div class="custom-listbox">
    <Listbox v-model="value">
      <ListboxButton class="custom-listbox-button">
        <span v-if="selectedItem" class="truncate">
          {{ t(`privileges.${selectedItem}`) }}
        </span>
        <span v-else class="truncate placeholder">{{ placeholder }}</span>
        <span class="custom-icon-container">
          <SvgIcon name="octicon-chevron-down" />
        </span>
      </ListboxButton>
      <ListboxOptions class="custom-listbox-options">
        <ListboxOption
          v-slot="{ active, selected }"
          v-for="item of items"
          :key="item"
          :value="item"
          as="template"
        >
          <li :class="[]" class="custom-list-item">
            <span
              :class="[selected ? 'custom-font-medium' : 'custom-font-normal']"
              class="truncate text-item"
            >
              {{ t(`privileges.${item}`) }}
            </span>
            <span v-if="selected" class="check-mark">
              <SvgIcon name="octicon-check" />
            </span>
          </li>
        </ListboxOption>
      </ListboxOptions>
    </Listbox>
  </div>
</template>

<script>
import {
  Listbox,
  ListboxButton,
  ListboxOptions,
  ListboxOption,
} from "@headlessui/vue";
import { SvgIcon } from "../../svg";
import { useI18n } from "vue-i18n";

export default {
  name: "PrivilegesDropDown",
  components: {
    Listbox,
    ListboxButton,
    ListboxOption,
    ListboxOptions,
    SvgIcon,
  },
  setup() {
    const { t, locale } = useI18n({
      inheritLocale: true,
      useScope: "local",
    });

    return { t, locale };
  },
  props: {
    selectedItem: String,
    placeholder: {
      type: String,
    },
    items: {
      type: Array,
      required: true,
    },
  },
  computed: {
    value: {
      get() {
        return this.selectedItem;
      },
      set(value) {
        this.$emit("update:selectedItem", value);
      },
    },
  },
};
</script>

<style scoped>
.custom-listbox {
  position: relative;
  width: 100%;
}

.custom-listbox-button {
  position: relative;
  width: 100%;
  display: flex;
  cursor: pointer;
  border-radius: 8px;
  background-color: var(--color-header-bar);
  padding: 10px 16px;
  border-style: solid;
  border-color: #d5d9dd;
  border-width: 1px;
}

.custom-listbox-button:active,
.custom-listbox-button:focus,
.custom-listbox-button:visited {
  border-color: rgb(1, 128, 255);
}

.truncate {
  display: block;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  min-width: 0px;
}

.truncate.placeholder {
  color: var(--color-text-light-2);
}

.custom-icon-container {
  pointer-events: none;
  position: absolute;
  right: 0;
  display: flex;
  align-items: center;
  padding-right: 0.5rem;
}

.custom-listbox-options {
  position: absolute;
  z-index: 11;
  margin-top: 0.25rem;
  max-height: 15rem;
  width: 100%;
  overflow-y: auto;
  border-radius: 0.375rem;
  background-color: var(--color-header-bar);
  padding: 0.25rem 0;
  font-size: 1rem;
  box-shadow: 0 1px 5px rgba(0, 0, 0, 0.05);
  border: 1px solid rgba(0, 0, 0, 0.1);
  outline: none;
}

.custom-list-item {
  display: flex;
  cursor: default;
  user-select: none;
  padding: 10px 16px;
  cursor: pointer;
}

.custom-list-item:hover,
.custom-list-item:focus {
  background-color: #f9f9f9;
  color: #3b3738;
}

.custom-font-medium {
  font-weight: 500;
}

.custom-font-normal {
  font-weight: 400;
}

.check-mark {
  width: 16px;
  aspect-ratio: 1/1;
  height: auto;
}

.check-mark svg {
  color: #1a5ee6;
}

.text-item {
  flex: 1;
}
</style>
