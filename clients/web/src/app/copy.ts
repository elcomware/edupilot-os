import type {Locale} from '@edupilot/api-client'
/** A required English/French pair makes untranslated additions a compile-time error. */
const strings = {
  brand:['School operations','Gestion scolaire'], signIn:['Sign in','Se connecter'], signingIn:['Signing in…','Connexion…'],
  access:['Staff access','Accès du personnel'], register:['The school register','Le registre scolaire'], loginInfo:['Sign in to open your school workspace.','Connectez-vous pour ouvrir votre espace scolaire.'],
  welcome:['Open your workspace','Ouvrir votre espace'], school:['School ID','Identifiant de l’établissement'],email:['Email','Adresse e-mail'], password:['Password','Mot de passe'],
  schoolHint:['Use the school ID provided by your administrator.','Utilisez l’identifiant fourni par votre administration.'],
  sessionExpired:['Your session ended. Sign in to continue.','Votre session est terminée. Reconnectez-vous pour continuer.'],
  invalidLogin:['Check your school ID, email and password.','Vérifiez votre identifiant d’établissement, votre adresse e-mail et votre mot de passe.'],
  workspace:['Workspace','Espace de travail'], directory:['Module directory','Répertoire des modules'], allModules:['All modules','Tous les modules'],
  directoryInfo:['Find a module or pin it to your navigation.','Trouvez un module ou épinglez-le à votre navigation.'],
  search:['Find a module','Rechercher un module'],group:['Group','Groupe'],allGroups:['All groups','Tous les groupes'],
  pinned:['Pinned modules','Modules épinglés'], noPins:['Pin a module from the directory.','Épinglez un module depuis le répertoire.'],
  pin:['Pin','Épingler'],unpin:['Unpin','Désépingler'], prefs:['Preferences','Préférences'], appearance:['Appearance','Apparence'],accent:['Accent','Couleur d’accentuation'],
  density:['Density','Densité'],width:['Reading width','Largeur de lecture'],language:['Language','Langue'],personal:['Personal settings','Réglages personnels'],
  prefInfo:['Choose how this workspace looks on this device.','Personnalisez l’apparence de cet espace sur cet appareil.'],
  appearanceInfo:['Use a light or dark background, or follow your device.','Choisissez un fond clair ou sombre, ou suivez le réglage de votre appareil.'],
  accentInfo:['Choose the color used for links and primary actions.','Choisissez la couleur des liens et des actions principales.'],
  densityInfo:['Adjust the space between rows.','Réglez l’espacement entre les lignes.'],widthInfo:['Adjust the maximum width of the content area.','Ajustez la largeur maximale de la zone de contenu.'],
  languageInfo:['Choose the language for labels and navigation.','Choisissez la langue des libellés et de la navigation.'],
  system:['System','Système'],light:['Light','Clair'],dark:['Dark','Sombre'],pine:['Pine','Pin'],indigo:['Indigo','Indigo'],clay:['Clay','Argile'],
  comfortable:['Comfortable','Aérée'],compact:['Compact','Compacte'],standard:['Standard','Standard'],wide:['Wide','Large'],
  prefsSaved:['Preferences saved on this device.','Préférences enregistrées sur cet appareil.'],storageUnavailable:['Preferences apply for this session only; device storage is unavailable.','Ces préférences s’appliquent à cette session ; le stockage local est indisponible.'],
  signOut:['Sign out','Se déconnecter'],menu:['Navigation','Navigation'],skip:['Skip to content','Aller au contenu'],
  loading:['Loading…','Chargement…'],retry:['Try again','Réessayer'],loadFailed:['Couldn’t load this workspace.','Impossible de charger cet espace.'],
  unavailable:['This page isn’t available.','Cette page est indisponible.'],denied:['You don’t have access to this workspace.','Vous n’avez pas accès à cet espace.'],
  network:['Couldn’t reach the server. Check your connection and try again.','Impossible de joindre le serveur. Vérifiez votre connexion et réessayez.'],
  stale:['This record changed. Review the latest version before saving.','Cette fiche a changé. Consultez la dernière version avant d’enregistrer.'],
  missing:['This record is no longer available.','Cette fiche n’est plus disponible.'],badInput:['Check the fields and try again.','Vérifiez les champs et réessayez.'],
  server:['The request could not be completed. Try again.','La demande n’a pas abouti. Réessayez.'],rate:['Too many attempts. Wait a moment and try again.','Trop de tentatives. Patientez un instant et réessayez.'],
  loadLatest:['Load latest version','Charger la dernière version'],emptyModules:['No modules available','Aucun module disponible'],emptyModulesHint:['Ask your administrator about workspace access.','Contactez votre administration au sujet de vos accès.'],
  noMatches:['No matching modules','Aucun module correspondant'],clearFilters:['Clear filters','Effacer les filtres'],
  students:['Students','Élèves'],studentRegister:['Student register','Registre des élèves'],studentInfo:['Manage student names and registration status.','Gérez les noms des élèves et leur statut d’inscription.'],
  entries:['Workspace entries','Entrées de l’espace'],entryInfo:['Keep track of entries in this module.','Suivez les entrées de ce module.'],
  addStudent:['Add student','Ajouter un élève'],editStudent:['Edit student','Modifier un élève'],archiveStudent:['Archive student','Archiver l’élève'],
  addEntry:['Add entry','Ajouter une entrée'],editEntry:['Edit entry','Modifier une entrée'],new:['New','Nouveau'],edit:['Edit','Modifier'],archive:['Archive','Archiver'],
  given:['Given name','Prénom'],family:['Family name','Nom de famille'],title:['Title','Titre'],status:['Status','Statut'],name:['Name','Nom'],
  active:['Active','Actif'],archived:['Archived','Archivé'],open:['Open','Ouverte'],closed:['Closed','Fermée'],
  saveStudent:['Save student','Enregistrer l’élève'],saveChanges:['Save changes','Enregistrer les modifications'],saveEntry:['Save entry','Enregistrer l’entrée'],
  cancel:['Cancel','Annuler'],saving:['Saving…','Enregistrement…'],confirmArchive:['Archive this student record?','Archiver cette fiche élève ?'],
  archiveNote:['The student will no longer be active.','L’élève ne sera plus actif.'],studentSaved:['Student saved.','Élève enregistré.'],entrySaved:['Workspace entry saved.','Entrée enregistrée.'],studentArchived:['Student archived.','Élève archivé.'],
  noStudents:['No students on this page','Aucun élève sur cette page'],noEntries:['No entries on this page','Aucune entrée sur cette page'],
  addFirstStudent:['Add a student to start the register.','Ajoutez un élève pour commencer le registre.'],addFirstEntry:['Add the first entry to this workspace.','Ajoutez une première entrée dans cet espace.'],
  actions:['Actions','Actions'],previous:['Previous','Précédent'],next:['Next','Suivant'],of:['of','sur'],records:['Records','Fiches'],back:['Back to directory','Retour au répertoire'],
  logoutUnconfirmed:['You signed out on this device. The server could not confirm revocation.','Vous êtes déconnecté sur cet appareil. Le serveur n’a pas confirmé la révocation.'],
  writeUnknown:['No confirmation received. Check the current records before trying again; the change may already be saved.','Aucune confirmation reçue. Vérifiez les fiches avant de réessayer : la modification peut déjà être enregistrée.'],
  checkList:['Return to list and refresh','Revenir à la liste et actualiser'],timeout:['The server took too long to respond. Try again.','Le serveur a mis trop de temps à répondre. Réessayez.'],
  offline:['You are offline. No changes were sent. Reconnect before trying again.','Vous êtes hors connexion. Aucune modification n’a été envoyée. Reconnectez-vous avant de réessayer.'],
  schoolRequired:['Enter your school ID.','Saisissez votre identifiant d’établissement.'],emailRequired:['Enter a valid email address.','Saisissez une adresse e-mail valide.'],passwordRequired:['Enter your password.','Saisissez votre mot de passe.'],
  nameRule:['Use 1–100 characters without control characters.','Utilisez de 1 à 100 caractères, sans caractère de contrôle.'],titleRule:['Use a title of 1–160 characters.','Utilisez un titre de 1 à 160 caractères.'],
  discard:['Discard changes?','Abandonner les modifications ?'],discardHint:['Closing this form discards the changes you haven’t saved.','Fermer ce formulaire abandonnera vos modifications non enregistrées.'],keepEditing:['Keep editing','Continuer la saisie'],discardAction:['Discard','Abandonner'],
  platform:['Core platform','Plateforme'],lifecycle:['Student lifecycle','Parcours des élèves'],academics:['Academic operations','Pédagogie'],finance:['Finance and commerce','Finances et commerce'],operations:['People and operations','Personnel et services'],engagement:['Engagement and intelligence','Communication et pilotage']
} as const satisfies Record<string, readonly [string,string]>
export type CopyKey = keyof typeof strings
export function t(locale: Locale, k: CopyKey): string {return strings[k][locale === 'fr' ? 1 : 0]}
export function groupLabel(locale: Locale, group: string) {
  const keys = ['platform','lifecycle','academics','finance','operations','engagement']
  return keys.includes(group) ? t(locale,group as CopyKey) : group
}
