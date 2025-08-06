<script>
import {SvgIcon} from '../../svg.js';
import {useI18n} from 'vue-i18n';
import PrivilegesDropDown from './PrivilegesDropDown.vue';
import PrivilegesRevokeDialog from './PrivilegesDialog.vue';
import PrivilegesTable from './PrivilegesTable.vue';
import EasyDataTable from 'vue3-easy-data-table';
import BaseButton from '../../ui/BaseButton.vue';

const {pageData, csrfToken} = window.config;
const {tenantId, orgId, enablePrivilegeManagement} = pageData;

export default {
  name: 'AppPrivileges',
  components: {
    SvgIcon,
    PrivilegesDropDown,
    PrivilegesTable,
    EasyDataTable,
    BaseButton,
    PrivilegesRevokeDialog,
  },
  setup() {
    const {t, locale} = useI18n({
      inheritLocale: true,
      useScope: 'local',
    });

    return {t, locale};
  },
  data() {
    return {
      enablePrivilegeManagement: enablePrivilegeManagement,
      currentUserPrivileges: {},
      isDialogOpen: false,
      currentUser: pageData.currentUser,
      isSidebarOpen: false,
      inputSearch: '',
      isFetched: false,
      users: [],
      roles: Object.values(pageData.roles),
      roleNames: Object.values(pageData.roleNames),
      baseLink: pageData.baseLink,
      selectedUser: null,
      selectedRole: null,
      isDropdownOpen: false,
      mainTableItems: pageData.priv?.map((item) => {
        item.User.Avatar = this.extractImgSrc(item.User.Avatar);
        return item;
      }),
      userIdsWithPrivileges: pageData.priv?.map((item) => item.User.ID),
      currentPage: 1,
      userPerPage: 10,
    };
  },
  computed: {
    isEdit() {
      if (!this.selectedUser) {
        return false;
      }

      return Boolean(this.userIdsWithPrivileges.includes(this.selectedUser.ID));
    },
    isSubmitDisabled() {
      return !this.selectedRole;
    },
    paginatedUsers() {
      const start = (this.currentPage - 1) * this.userPerPage;
      const end = start + this.userPerPage;
      return this.users.slice(start, end);
    },
    totalPages() {
      const totalUsers = this.users?.length;
      const pages = Math.ceil(totalUsers / this.userPerPage);
      return pages > 0 ? pages : 1;
    },
    paginationPages() {
      const total = this.totalPages;
      const current = this.currentPage;
      const pages = [];

      if (total <= 5) {
        for (let i = 1; i <= total; i++) {
          pages.push(i);
        }
      } else {
        if (current <= 3) {
          pages.push(1, 2, 3, 4, '...', total);
        } else if (current >= total - 2) {
          pages.push(1, '...', total - 3, total - 2, total - 1, total);
        } else {
          pages.push(1, '...', current - 1, current, current + 1, '...', total);
        }
      }
      return pages;
    },
  },
  watch: {
    async isSidebarOpen(val) {
      if (val) {
        document.documentElement.style.overflow = 'hidden';
      } else {
        document.documentElement.style.overflow = '';
      }
      if (this.isFetched) {}
    },
  },
  mounted() {
    this.fetchPrivileges();
  },
  methods: {
    async fetchPrivileges() {
      const formData = new FormData();

      formData.append('user_id', this.currentUser.ID);
      formData.append('tenant_id', tenantId);
      formData.append('org_id', orgId);
      formData.append('_csrf', csrfToken);

      const response = await fetch(`${this.baseLink}/user`, {
        method: 'post',
        body: formData,
      });
      const data = await response.json();
      for (const item of data.privileges) {
        this.currentUserPrivileges[item] = true;
      }
    },
    toggleDropdown() {
      this.isDropdownOpen = !this.isDropdownOpen;
    },
    toggleSidebar() {
      this.isSidebarOpen = !this.isSidebarOpen;

      if (!this.isSidebarOpen) {
        this.selectedRole = null;
        this.selectedUser = null;
      }

      this.inputSearch = '';
      this.users = [];
    },
    async fetchUsers(inputSearch) {
      const response = await fetch(`${this.baseLink}/grant?search=${inputSearch}`, {method: 'get'});
      const data = await response.json();
      if (data.users) {
        this.users = data.users?.map((user) => {
          user.Avatar = this.extractImgSrc(user.Avatar);
          return user;
        });
      }
    },

    async handleSearchUser(inputSearch) {
      await this.fetchUsers(inputSearch);
    },

    handleSelectUserPrivileges(userPrivileges, role = null) {
      this.selectedUser = userPrivileges;
      this.selectedRole = role;
    },
    handleResetSelectedUserPrivileges() {
      this.selectedUser = null;
      this.selectedRole = null;
    },
    async handleSubmit() {
      const formData = new FormData();
      formData.append('user_id', this.selectedUser.ID);
      formData.append('tenant_id', tenantId);
      formData.append('org_id', orgId);
      formData.append('role', this.selectedRole);
      formData.append('_csrf', csrfToken);

      const response = await fetch(`${this.baseLink}/grant`, {
        method: 'POST',
        body: formData,
      });

      if (!response.ok) {}

      window.location.reload();
    },
    async handleRevokePrivileges() {
      const formData = new FormData();
      formData.append('user_id', this.selectedUser.ID);
      formData.append('tenant_id', tenantId);
      formData.append('org_id', orgId);
      formData.append('role', this.selectedRole);
      formData.append('_csrf', csrfToken);

      const response = await fetch(`${this.baseLink}/revoke`, {
        method: 'POST',
        body: formData,
      });

      if (!response.ok) {}

      window.location.reload();
    },
    onEditClick(user, role) {
      this.toggleSidebar();
      this.handleSelectUserPrivileges(user, role);
    },
    onRowClick(row) {
      this.handleSelectUserPrivileges(row);
    },
    extractImgSrc(str) {
      return str.match(/src="([^"]*)"/)[1];
    },
    goToPage(page) {
      if (page !== '...' && page >= 1 && page <= this.totalPages) {
        this.currentPage = page;
      }
    },
  },
};
</script>

<template>
  <div class="test">
    <div class="main content">
      <button
        v-if="currentUserPrivileges.own && enablePrivilegeManagement"
        @click="toggleSidebar()"
        class="ui primary button"
      >
        <svg-icon name="octicon-plus" />
        {{ t("privileges.addButton") }}
      </button>
      <PrivilegesTable
        :roles="roles"
        :roleNames="roleNames"
        :privileges="mainTableItems"
        @onEditClick="onEditClick"
        :is-edit-able="currentUserPrivileges.own && enablePrivilegeManagement"
        :current-user="currentUser"
      />
    </div>
    <div v-if="isSidebarOpen" class="visible ui right sidebar vertical menu">
      <div class="sidebar-inner-wrapper">
        <div class="sidebar-header">
          <h2 v-if="isEdit">
            {{ t("privileges.editingMember") }}
          </h2>
          <h2 v-else>
            {{ t("privileges.addingMember") }}
          </h2>
          <base-button type="transparent" icon @click="toggleSidebar()">
            <svg-icon name="octicon-x" />
          </base-button>
        </div>
        <template v-if="!selectedUser">
          <div class="sidebar-content">
            <div class="sidebar-search">
              <input
                type="text"
                class="sidebar-search__field"
                v-model="inputSearch"
                :placeholder="t('privileges.searchPlaceholder')"
                @input="handleSearchUser(inputSearch)"
              >
              <svg-icon class="sidebar-search__icon" name="octicon-search" />
            </div>
            <div class="sidebar-content_users">
              <easy-data-table
                v-if="users.length || inputSearch !== ''"
                :headers="[
                  {
                    text: t('privileges.table.header.name'),
                    value: 'name',
                    sortable: true,
                  },
                ]"
                :items="inputSearch !== '' ? paginatedUsers : []"
                theme-color="#1A5EE6"
                :hide-footer="true"
                alternating
                table-class-name="users-table"
                @click-row="onRowClick"
                body-row-class-name="users-table_row"
              >
                <template #item-name="user">
                  <div class="user-item">
                    <div class="img-wrapper">
                      <img
                        :src="user.Avatar"
                        alt=""
                        class="ui avatar gt-vm tiny"
                      >
                    </div>
                    <div>
                      {{ user.FullName }}
                      {{ user.Name }}
                    </div>
                  </div>
                </template>
                <template #empty-message>
                  <div class="no-data">
                    <svg-icon class="no-users-img" name="no-users"/>
                    <p class="no-user-text">
                      {{ t('noUser') }}
                    </p>
                  </div>
                </template>
              </easy-data-table>
              <div v-else class="no-data">
                <svg-icon class="no-users-img" name="no-users"/>
                <p class="no-user-text">
                  {{ t('noData') }}
                </p>
              </div>
              <div v-if="users.length !== 0 && users.length > 10 && inputSearch !== ''" class="pagination">
                <button
                  v-for="page in paginationPages"
                  :key="page"
                  @click="goToPage(page)"
                  :class="{ active: currentPage === page }"
                  :disabled="page === '...'"
                >
                  {{ page }}
                </button>
              </div>
            </div>
          </div>
        </template>
        <form v-else class="selected-user" @submit.prevent="handleSubmit">
          <div class="user">
            <div class="img-wrapper">
              <img
                :src="selectedUser.Avatar + '&size=100'"
                alt="аватар пользователя"
                width="100"
                class="ui avatar gt-vm"
              >
            </div>
            <div class="name-email">
              <p>
                {{ selectedUser.LowerName }}
              </p>
              <a :href="'mailto:' + selectedUser.Email">
                {{ selectedUser.Email }}
              </a>
            </div>
          </div>
          <div>
            <PrivilegesDropDown
              :items="roles"
              :roleNames="roleNames"
              :selected-item="selectedRole"
              @update:selectedItem="(value) => (selectedRole = value)"
              :placeholder="t('privileges.select')"
            />
          </div>
          <div class="actions">
            <button
              type="submit"
              class="button ui primary"
              :disabled="isSubmitDisabled"
            >
              {{ isEdit ? t("buttons.save") : t("buttons.add") }}
            </button>
            <button
              class="button ui"
              @click="handleResetSelectedUserPrivileges()"
            >
              {{ t("buttons.cancel") }}
            </button>
            <button
              type="button"
              v-if="isEdit && currentUserPrivileges.own"
              class="button ui delete"
              @click="isDialogOpen = true"
            >
              <svg-icon name="octicon-trash" />
            </button>
          </div>
        </form>
      </div>
    </div>
    <div v-if="isSidebarOpen" @click="toggleSidebar()" class="overlay"/>
    <PrivilegesRevokeDialog
      v-if="isDialogOpen"
      @on-close="isDialogOpen = false"
      @onConfirm="handleRevokePrivileges"
    />
  </div>
</template>

<style scoped>
.sidebar.visible.menu {
  background-color: var(--color-body) !important;
}

.sidebar-content_users {
  height: 75vh;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.sidebar-content {
  height: 100%;
  display: flex;
  flex-direction: column;
  flex-grow: 1;
}

.users-table {
  --easy-table-border: none;
  --easy-table-body-row-height: 50px;
  --easy-table-body-row-font-color: #737b8c;
  --easy-table-body-even-row-font-color: #737b8c;
  --easy-table-header-font-size: 13px;
  --easy-table-body-row-font-size: 13px;
  --easy-table-body-row-hover-font-color: inherit;
  --easy-table-loading-mask-background-color: var(--color-primary);
  --easy-table-body-margin-block-end: 10px;
}

.theme-dark .users-table {
  --easy-table-header-font-color: var(--color-caret);
  --easy-table-header-background-color: var(--color-body);

  --easy-table-body-row-background-color: var(--color-body);
  --easy-table-body-even-row-background-color: var(--color-body);
  --easy-table-body-even-row-font-color: var(--color-text);
  --easy-table-body-row-font-color: var(--color-text);

  --easy-table-body-row-hover-font-color: #737b8c;
  --easy-table-body-row-hover-background-color: #282e3f;
  --easy-table-body-row-even-hover-font-color: #737b8c;
  --easy-table-body-row-even-hover-background-color: #282e3f;
}

.users-table_row {
  cursor: pointer;
}

button {
  display: flex !important;
  align-items: center;
  gap: 8px;
}

.user-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 8px 14px;
}

.user-item .img-wrapper {
  aspect-ratio: 1/1;
  width: 48px;
  height: 48px;
  border-radius: 8px;
}

.user-item .img-wrapper img {
  width: 100%;
}

.sidebar {
  position: fixed;
  right: 0 !important;
  bottom: 0 !important;
  width: 443px !important;
  height: 100vh !important;
  background-color: white !important;
  z-index: 11;
  margin: 0px !important;
  border-radius: 0px !important;
}

.overlay {
  position: fixed;
  right: 0 !important;
  bottom: 0 !important;
  left: 0 !important;
  top: 0 !important;
  background-color: rgba(0, 0, 0, 0.4);
  z-index: 10;
  overflow: hidden;
}

.sidebar-inner-wrapper {
  padding: 32px;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.sidebar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.sidebar-header h2 {
  margin: 0px;
}

.ui.small.icon.button.pencil {
  background-color: transparent;
  border: none;
}

.main.content > :not(:last-child) {
  margin-bottom: 10px;
}

.selected-user {
  display: flex;
  flex-direction: column;
  flex-grow: 1
}

.selected-user .user {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 16px;
  height: 100px;
  width: 100%;
  margin-bottom: 24px;
}

.selected-user .user .name-email {
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.selected-user .user .name-email > * {
  margin: 0;
}

.selected-user .user .img-wrapper {
  aspect-ratio: 1/1;
  width: 100px;
  height: 100px;
  border-radius: 8px;
}

.selected-user .actions {
  display: flex;
  gap: 16px;
  margin-top: auto;
}

.selected-user .actions button.delete {
  background-color: red;
  flex: 0 !important;
  color: white;
}

.selected-user .actions > * {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0px !important;
}

.sidebar-search {
  display: flex;
  align-items: center;
  margin: 24px 0;
  width: 100%;
  height: 40px;
  border: 1px solid #D5D9DD;
  padding: 10px 16px;
  border-radius: 8px;
  background: transparent;
  column-gap: 12px;
}

.sidebar-search:focus-within {
  border-width: 2px;
  border-color:  #1A5EE6;
}

.sidebar-search:focus-within .sidebar-search__icon {
  color: #1A5EE6;
}

.sidebar-search__field {
  flex-grow: 1;
  border: none;
  background: transparent;
  outline: none;
}

.sidebar-search__icon {
  flex-shrink: 0;
  color: #2E3038;
}

.theme-dark  .sidebar-search__icon {
  color: var(--color-caret)
}

.no-user-text {
  max-width: 370px;
  font-size: 15px;
  white-space: pre-line;
}

.no-data {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  text-align: center;
  white-space: pre-line;
}

.no-users-img {
  width: 150px;
  height: auto;
}

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 5px;
  margin-top: auto;
}

.pagination button {
  padding: 5px 10px;
  cursor: pointer;
  background-color: transparent;
  border: 1px solid #ccc;
  border-radius: 3px;
}

.pagination button.active {
  background-color: #1A5EE6;
  color: white;
  border-color: #1A5EE6;
}

.pagination button:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}
</style>

<i18n>
{
  "en-US": {
    "noData": "Users list is empty",
    "noUser": "User is not found",
    "owner": "Owner",
    "select": "Select group of privileges",
  },
  "ru-RU": {
    "noData": "Список пользователей пуст",
    "noUser": "Пользователь не найден,\n попробуйте добавить другого участника",
    "owner": "Владелец проекта",
    "select": "Выберите группу привилегий",
  }
}
</i18n>
