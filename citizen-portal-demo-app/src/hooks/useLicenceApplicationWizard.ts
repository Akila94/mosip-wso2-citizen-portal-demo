import { useState } from 'react';
import type { DeclarationAnswer } from '../services/types';

export interface UploadedDoc {
  name: string;
  sizeLabel: string;
}

/**
 * Holds the in-progress application's interactive state across all four
 * steps. In-memory only for this demo (per product decision) — "Save &
 * exit" navigates away without persisting a draft. Reference data (licence
 * classes, fee schedule, etc.) comes from applicationService via
 * useApplicationConfig; this hook owns only what the citizen has entered.
 */
export function useLicenceApplicationWizard() {
  const [appTypeId, setAppTypeId] = useState('new');
  const [selectedClassIds, setSelectedClassIds] = useState<string[]>(['b', 'c1']);
  const [fieldValues, setFieldValues] = useState<Record<string, string>>({});
  const [collectionDistrict, setCollectionDistrict] = useState('Marolia Central');
  const [pickupStation, setPickupStation] = useState('Marolia West licence office');
  const [postInstead, setPostInstead] = useState(false);
  const [declarationAnswers, setDeclarationAnswers] = useState<Record<string, DeclarationAnswer>>({ vision: 'yes', colourBlind: 'no', blackouts: null });
  const [medicalCert] = useState<UploadedDoc | null>({ name: 'medical-fitness-aug2026.pdf', sizeLabel: '1.2 MB · issued 02 Aug 2026 · checks passed' });
  const [learnerPermitDoc, setLearnerPermitDoc] = useState<UploadedDoc | null>(null);
  const [paymentMethod, setPaymentMethod] = useState<'now' | 'later'>('now');
  const [digitalSignature, setDigitalSignature] = useState(false);
  const [declarationAgreed, setDeclarationAgreed] = useState(false);

  function toggleClass(id: string) {
    setSelectedClassIds((prev) => (prev.includes(id) ? prev.filter((c) => c !== id) : [...prev, id]));
  }

  function setDeclaration(id: string, answer: DeclarationAnswer) {
    setDeclarationAnswers((prev) => ({ ...prev, [id]: answer }));
  }

  function setField(id: string, value: string) {
    setFieldValues((prev) => ({ ...prev, [id]: value }));
  }

  return {
    appTypeId, setAppTypeId,
    selectedClassIds, toggleClass,
    fieldValues, setField,
    collectionDistrict, setCollectionDistrict,
    pickupStation, setPickupStation,
    postInstead, setPostInstead,
    declarationAnswers, setDeclaration,
    medicalCert,
    learnerPermitDoc, setLearnerPermitDoc,
    paymentMethod, setPaymentMethod,
    digitalSignature, setDigitalSignature,
    declarationAgreed, setDeclarationAgreed,
    triggersMedicalReview: declarationAnswers.blackouts === 'yes',
  };
}

export type LicenceApplicationWizardState = ReturnType<typeof useLicenceApplicationWizard>;
