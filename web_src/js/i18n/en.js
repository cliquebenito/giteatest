export const enDict = {
  buttons: {
    cancel: "Cancel",
    clear: "Clear",
    reset: "Reset",
    save: "Save",
    submit: "Submit",
    add: "Add",
    delete: "Delete",
  },
  license: {
    permission: "Permission",
    limitations: "Limitations",
    conditions: "Conditions",
  },


  yes: "Yes",
  no: "No",

  privileges: {
    delete: {
      title: "Removing",
      message: "Are you sure you want to remove the user from the project ?",
    },
    addButton: "Add member",
    addingMember: "Adding a member",
    editingMember: "Editing",
    select: "Select group of privileges",
    table: {
      header: {
        name: "Name and login",
        role: "Group of privileges in the project",
        edit: "Edit",
      },
      emptyMessage: "No data",
    },
    searchPlaceholder: "Search...",
    owner: "Owner",
    reader: "Reader",
    writer: "Writer",
    manager: "Manager",
  },

  tenant: {
    placeholder: {
      search: "Search tenants",
      create: "Tenant name",
      edit: "Tenant name",
    },

    isActiveLabel: "Tenant active",

    createButton: "Create tenant",

    title: {
      page: "Tenants",
      create: "Create",
      edit: "Edit",
      projects: "Assigned projects",
      delete: "Delete",
    },

    tooltip: {
      edit: "Edit tenant",
      delete: "Delete tenant",
    },

    tableTenantsHeaders: {
      name: "Name",
      projects: "Projects",
      created: "Created At",
      edit: "Edit",
    },

    emptyTable: "No data by filter",

    tableProjectsHeaders: {
      name: "Name",
      create: "Create At",
    },

    hint: "If the tenant is not active, then all related projects and repositories will no longer be displayed on the platform",

    emptyProjects: "There is no projects in current tenant",

    deleteWarning:
      "Are you sure you want to delete {name}? All related projects and repositories will also be deleted",

    message: {
      success: {
        delete: "Tenant - {name} deleted",
        create: "Tenant - {name} created",
        update: "Tenant - {name} updated",
      },
      failure: {
        delete: "Can't delete tenant. {error}",
        create: "Can't create tenant. {error}",
        update: "Can't update tenant. {error}",
      },
    },
  },
};
