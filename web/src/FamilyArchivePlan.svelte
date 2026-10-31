<script lang="ts">
 import type {FamilyPlan} from './family-archive';
 export let plan:FamilyPlan;
 const labels:Record<string,string>={install:'Installer · en pause',update:'Mettre à jour',reuse:'Réutiliser',keep:'Conserver le composant choisi'};
</script>
 <ul>{#each plan.steps as step}<li><strong>{labels[step.operation]}</strong> · {step.operation==='keep'?step.previousId:step.name}{step.operation==='keep'?'':' · '+step.version}{step.shared?' · partagé':''}{step.select?' · devient le composant choisi':''}</li>{/each}</ul>
 <p>Les nouveaux composants restent en pause. Une mise à jour conserve les états existants. Un nouvel écran remplace le choix de l’écran fourni et met celui-ci en pause ; ses accès utilisateurs seront à définir dans les réglages avancés.</p>
 <p>Programmes isolés : pas d’accès réseau. Les composants musicaux peuvent lire les données du fichier analysé. Aucun accès général aux fichiers ni aux comptes.</p>
 <p>Tout est installé ensemble ou rien ne change. Activez ensuite la famille depuis sa fiche.</p>
