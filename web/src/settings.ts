// Host-owned navigation entries. Later extension contributions use this same
// shape after permission/dependency validation; this is not an installation API.
export type SettingsSection={id:string;label:string;description:string;video?:boolean;admin?:boolean};
export const settingsSections:SettingsSection[]=[
 {id:'account',label:'Mon compte',description:'Mot de passe et apparence de votre espace.'},
 {id:'users',label:'Utilisateurs',description:'Comptes, invitations et permissions.',admin:true},
 {id:'extensions',label:'Extensions',description:'Paquets installés et thèmes personnels.'},
 {admin:true,id:'libraries',label:'Bibliothèques',description:'Les dossiers qui composent votre collection.'},
 {admin:true,id:'metadata',label:'Métadonnées',description:'Fournisseurs, affiches et suivi de l’identification.'},
 {admin:true,id:'downloads',label:'Téléchargements',description:'Vos services et les destinations de vos demandes.',video:true},
 {admin:true,id:'server',label:'Serveur',description:'Informations de cette installation et essais ponctuels.'},
];
