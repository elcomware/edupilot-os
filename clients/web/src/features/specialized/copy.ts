import type {Locale} from '@edupilot/api-client'
const pairs={
 placements:['Enrollment placements','Inscriptions'], placementHelp:['Assign students to a grade for an academic year.','Inscrivez les élèves à un niveau pour une année scolaire.'],
 add:['Add placement','Ajouter une inscription'], year:['Academic year','Année scolaire'], yearHint:['Use two consecutive years, for example 2026-2027.','Saisissez deux années consécutives, par exemple 2026-2027.'],
 grade:['Grade level','Niveau scolaire'],student:['Student','Élève'],choose:['Choose a student','Choisir un élève'],studentPage:['Student list','Liste des élèves'],
 placed:['Placement saved.','Inscription enregistrée.'],withdraw:['Withdraw placement','Retirer l’inscription'],withdrawn:['Withdrawn','Retirée'],withdrawnNote:['Placement withdrawn.','Inscription retirée.'],
 withdrawHint:['Withdrawing ends this placement for the selected academic year.','Ce retrait met fin à l’inscription pour l’année scolaire sélectionnée.'],
 empty:['No placements on this page','Aucune inscription sur cette page'],nextStudents:['Next students','Élèves suivants'],prevStudents:['Previous students','Élèves précédents'],
 noActive:['No active students are available.','Aucun élève actif disponible.'],gradeRule:['Enter a grade of 1–48 characters.','Saisissez un niveau de 1 à 48 caractères.'],
 enrollmentConflict:['The student or placement changed. Review the list before trying again.','L’élève ou son inscription a changé. Vérifiez la liste avant de réessayer.'],
 documents:['Document files','Fichiers de l’établissement'], docsHelp:['Upload a file or retrieve one using its document ID.','Ajoutez un fichier ou retrouvez-le avec son identifiant.'],
 upload:['Upload file','Importer le fichier'],uploading:['Uploading…','Importation…'],file:['File','Fichier'],fileHelp:['PDF, plain text, PNG or JPEG. Maximum 8 MiB.','PDF, texte brut, PNG ou JPEG. Maximum : 8 Mio.'],
 chooseFile:['Choose a nonempty supported file up to 8 MiB.','Choisissez un fichier pris en charge et non vide de 8 Mio maximum.'],
 download:['Download file','Télécharger le fichier'],downloading:['Downloading…','Téléchargement…'],id:['Document ID','Identifiant du document'],
 idHint:['Enter the ID returned when the document was uploaded.','Saisissez l’identifiant fourni lors de l’importation.'],idRule:['Enter a valid document ID.','Saisissez un identifiant de document valide.'],
 uploaded:['File uploaded. Keep this document ID for later.','Fichier importé. Conservez cet identifiant pour le retrouver.'],downloaded:['Download started.','Téléchargement lancé.'],
 uploadUnknown:['No upload confirmation was received. The file may exist without a returned ID. Check with your administrator before uploading it again.','L’importation n’a pas été confirmée. Le fichier peut exister sans identifiant reçu. Consultez votre administration avant de réessayer.'],
 studentRequired:['Choose an active student.','Choisissez un élève actif.'], unavailable:['File storage is unavailable. Try again later.','Le stockage des fichiers est indisponible. Réessayez plus tard.'],unsupported:['The file contents or format are not supported.','Le contenu ou le format du fichier n’est pas pris en charge.']
} as const satisfies Record<string,readonly [string,string]>
export function sx(l:Locale,k:keyof typeof pairs):string{return pairs[k][l==='fr'?1:0]}
