<template>
  <easy-data-table
    :headers="headers"
    :items="items"
    theme-color="#1A5EE6"
    :hide-footer="true"
    table-class-name="privileges-table"
    body-row-class-name="row"
    body-item-class-name="row"
    alternating
    @update-sort="updateSort"
  >
    <template #item-name="{ User }">
      <div class="user">
        <div class="img-wrapper">
          <img
            :src="User.Avatar"
            alt=""
            class="ui avatar gt-vm tiny"
            width="48"
          />
        </div>
        <div>
          <p class="user-fullname">{{ User.FullName }}</p>
          <p>{{ User.Name }}</p>
        </div>
      </div>
    </template>
    <template #item-role="{ Role }">
      <div class="user-role">
        {{getRoleNameLabel(Role)}}
        <template v-if="Role === 'owner'">
          <SvgIcon name="crown" />
        </template>
      </div>
    </template>
    <template #item-edit="{ User, Role }">
      <button
        :disabled="!isEditAble || currentUser.ID === User.ID"
        class="ui small icon button pencil"
        @click="$emit('onEditClick', User, Role)"
      >
        <SvgIcon name="octicon-pencil" /></button
    ></template>

    <template #empty-message>
      <p class="empty-message">{{ $t("privileges.table.emptyMessage") }}</p>
    </template>
  </easy-data-table>
</template>

<script>
import EasyDataTable from "vue3-easy-data-table";
import { useI18n } from "vue-i18n";
import "vue3-easy-data-table/dist/style.css";
import { SvgIcon } from "../../svg";

export default {
  name: "PrivilegesTable",
  components: {
    EasyDataTable,
    SvgIcon,
  },
  props: {
    privileges: {
      type: Array,
      required: true,
    },
    isEditAble: Boolean,
    currentUser: Object,
    roles: {
      type: Array,
      required: true
    },
    roleNames: {
      type: Array,
      required: true
    },
  },
  emits: ["onEditClick"],
  data() {
    return {
      headers: [
        {
          text: this.t("privileges.table.header.name"),
          value: "name",
          sortable: true,
        },
        {
          text: this.t("privileges.table.header.role"),
          value: "Role",
          sortable: true,
        },
      ],
      sortBy: null,
      sortType: null,
    };
  },
  watch: {
    isEditAble(val) {
      if (val) {
        this.headers.push({
          text: this.t("privileges.table.header.edit"),
          value: "edit",
          width: 100,
        });
      }
    },
  },
  methods: {
    updateSort({ sortType, sortBy }) {
      this.sortBy = sortBy;
      this.sortType = sortType;
    },
    sortByName() {
      if (this.sortType === "desc") {
        return [...this.$props.privileges].sort(
          (a, b) => -1 * a.User.Name.localeCompare(b.User.Name)
        );
      } else if (this.sortType === "asc") {
        return [...this.$props.privileges].sort((a, b) =>
          a.User.Name.localeCompare(b.User.Name)
        );
      }

      return this.$props.privileges;
    },
    sortByRole() {
      if (this.sortType === "desc") {
        return [...this.$props.privileges].sort(
          (a, b) => -1 * a.Role.localeCompare(b.Role)
        );
      } else if (this.sortType === "asc") {
        return [...this.$props.privileges].sort((a, b) =>
          a.Role.localeCompare(b.Role)
        );
      }

      return this.$props.privileges;
    },
     getRoleNameLabel(rolename) {
      const targetIndex = Object.entries(this.$props.roles).find(([index, name]) => rolename === name)[0];
      return this.$props.roleNames[targetIndex]
    },
  },
  computed: {
    items() {
      if (this.sortBy === "name") {
        return this.sortByName();
      } else if (this.sortBy === "role") {
        return this.sortByRole();
      }

      return this.$props.privileges;
    },
  },
  setup() {
    const { t, locale } = useI18n();

    return { t, locale };
  },
};
</script>

<style scoped>
.user {
  display: flex;
  gap: 16px;
}

.user-role {
  display: flex;
  align-items: center;
  gap: 6px;
}

.user p {
  display: block;
}
.img-wrapper {
  aspect-ratio: 1/1;
  width: 48px;
  height: 48px;
  border-radius: 8px;
}
.privileges-table {
  --easy-table-border: none;
  --easy-table-body-row-height: 50px;
  --easy-table-body-row-font-color: #737b8c;
  --easy-table-body-even-row-font-color: #737b8c;
  --easy-table-header-font-size: 13px;
  --easy-table-body-row-font-size: 13px;
  --easy-table-body-row-hover-font-color: inherit;
  --easy-table-loading-mask-background-color: var(--color-primary);
  --easy-table-body-item-padding: 16px 8px;
}

.theme-dark .privileges-table {
  --easy-table-header-font-color: var(--color-caret);
  --easy-table-header-background-color: var(--color-body);

  --easy-table-body-row-background-color: var(--color-body);
  --easy-table-body-even-row-background-color: var(--color-header-bar);
  --easy-table-body-even-row-font-color: var(--color-text);
  --easy-table-body-row-font-color: var(--color-text);

  --easy-table-body-row-hover-font-color: #737b8c;
  --easy-table-body-row-hover-background-color: #282e3f;
  --easy-table-body-row-even-hover-font-color: #737b8c;
  --easy-table-body-row-even-hover-background-color: #282e3f;
  --easy-table-body-item-padding: 16px 8px;
}

.ui.small.icon.button.pencil {
  background-color: transparent;
  border: none;
  color: #1a5ee6;
}

.ui.small.icon.button.pencil:disabled {
  color: var(--color-text-light-2);
}

.privileges-table .row {
  padding-top: 16px;
  padding-bottom: 16px;
}

.user-fullname {
  color: var(--color-text);
  font-size: 15px;
}
</style>
