import * as _ from 'lodash-es';
import * as React from 'react';
import { match as RMatch } from 'react-router';
import { useFormContext, Controller } from 'react-hook-form';
import { WithCommonForm } from './form/create-form';
import { SelectorInput } from '../utils';
import { RadioGroup } from './utils/radio';
import { Section } from './utils/section';
import { InputSelectBox } from './utils/inputSelectBox';
import { Dropdown, ContainerDropdown } from './utils/dropdown';
import { ResourceDropdown } from './utils/resource-dropdown';
import { ResourceListDropdown, ResourceListDropdownWithDataToolbar } from './utils/resource-list-dropdown';
import { KeyValueListEditor } from './utils/key-value-list-editor';
import { TagsLabel } from './utils/tags-label';
import { NumberSpinner } from './utils/number-spinner';
import { ListView } from './utils/list-view';
import { Button } from '@patternfly/react-core';

const defaultValues = {
  // requestDo에 넣어줄 형식으로 defaultValues 작성
  metadata: {
    name: 'test-name',
    keyvaluelist: [
      { key: 'A', value: 'aaa' },
      { key: 'B', value: 'bbb' },
      { key: 'C', value: 'ccc' },
      { key: 'D', value: 'ddd' },
      { key: 'E', value: 'eee' },
    ],
    tags: ['AAA', 'BBB'],
  },
  spec: {
    resources: 'cpu',
  },
  keyValueList: [
    {
      key: 'AAA',
      value: 'aaa',
    },
    {
      key: 'BBB',
      value: 'bbb',
    },
    {
      key: 'CCC',
      value: 'ccc',
    },
    {
      key: 'DDD',
      value: 'ddd',
    },
  ],
  numList: [
    {
      name: 'Item1',
      number: 3,
    },
    {
      name: 'Item2',
      number: 5,
    },
  ],
  dropdown1: 'Ti',
};

const ClusterResourceList = [
  {
    kind: 'ClusterManager',
    apiVersion: 'cluster.tmax.io/v1alpha1',
    metadata: {
      name: 'jmc-zgw2v',
      uid: '1a482d7d-ac35-46d3-8496-a94688fc6d0e',
    },
    fakeMetadata: {
      fakename: 'jmc',
    },
  },
  {
    kind: 'ClusterClaim',
    apiVersion: 'cluster.tmax.io/v1alpha1',
    metadata: {
      name: 'example',
      uid: '436b6e22-748e-4e04-aea5-156f2ed35fa0',
    },
  },
];

const sampleFormFactory = params => {
  return WithCommonForm(CreateSampleComponent, params, defaultValues);
};

const CreateSampleComponent: React.FC<SampleFormProps> = props => {
  const methods = useFormContext();

  const updateSelectedClusterItems = (selection: string) => {
    // selection: {resource}.metadata.uid | 'All'
    //Do Something
    console.log('updateSelectedClusterItems: ', selection);
  };

  const onSelectedItemChange = (items: Set<string>) => {
    //DO Something
    console.log('hi');
  };

  const resources = [
    // RadioGroup 컴포넌트에 넣어줄 items
    {
      title: 'Cpu',
      value: 'cpu',
    },
    {
      title: 'Gpu',
      value: 'gpu',
    },
    {
      title: 'Memory',
      value: 'memory',
    },
  ];
  const dropdownUnits = {
    Mi: 'MiB',
    Gi: 'GiB',
    Ti: 'TiB',
  };
  const containers = { test: { name: 'test', order: 0 }, sidecar: { name: 'sidecar' }, sidecar2: { name: 'sidecar2' } };
  const initContainers = { initupload: { name: 'initupload', order: 0 }, ['place-entrypoint']: { name: 'place-entrypoint' }, ['place-entrypoint2']: { name: 'place-entrypoint2' } };

  const listHeaderFragment = (
    <div className="row pairs-list__heading">
      <div className="col-xs-4 text-secondary text-uppercase">NAME</div>
      <div className="col-xs-4 text-secondary text-uppercase">NUM</div>
      <div className="col-xs-1 co-empty__header" />
    </div>
  );

  const listItemRenderer = (register, name, item, index, ListActions, ListDefaultIcons) => (
    <div className="row" key={item.id}>
      <div className="col-xs-4 pairs-list__name-field">
        <input ref={register()} className="pf-c-form-control" name={`metadata.spinnerNumList[${index}].name`} defaultValue={item.name}></input>
      </div>
      <div className="col-xs-4 pairs-list__value-field">
        <NumberSpinner initialValue={item.number} min={-15} max={15} name={`metadata.spinnerNumList[${index}].number`} />
      </div>
      <div className="col-xs-1 pairs-list__action">
        <Button
          type="button"
          data-test-id="pairs-list__delete-btn"
          className="pairs-list__span-btns"
          onClick={() => {
            ListActions.remove(index);
          }}
          variant="plain"
        >
          {ListDefaultIcons.deleteIcon}
        </Button>
      </div>
    </div>
  );

  return (
    <div>
      <Section label="Labels" id="label" description="이것은 Label입니다.">
        <Controller name="metadata.labels" id="label" labelClassName="co-text-sample" as={SelectorInput} control={methods.control} tags={[]} />
      </Section>
      <Section id="resources" label="Radio Group">
        <RadioGroup
          name="spec.resources" // 서버에 보낼 데이터에서의 path (필수)
          items={resources} // [{title: '', value: ''}] (필수)
          inline={false} // inline속성 먹일거면 true, 아니면 빼면 됨 (선택)
        />
      </Section>
      <Section id="cpu" label="Input Selectbox">
        <InputSelectBox textName="spec.cpu" id="cpu" dropdownName="spec.cpuRange" selectedKey="Mi" items={dropdownUnits} />
      </Section>
      <Section id="section" label="Grid Section" isRequired={true}>
        {/* sample로 각각다른 3개 node 넣어봄. 1,2,3,4 개 일 경우 다 정상동작 하는 것 확인.*/}
        <Section id="label" label="Label (for Section)">
          <Controller name="metadata.section.label" id="label" labelClassName="co-text-sample" as={SelectorInput} control={methods.control} tags={[]} />
        </Section>
        <Section id="cpu" label="Input Selectbox (for Section)">
          <InputSelectBox textName="spec.section.cpu" id="cpu" dropdownName="spec.section.cpuRange" selectedKey="Mi" items={dropdownUnits} />
        </Section>
        <Section id="resources" label="Radio Group (for Section)">
          <RadioGroup
            name="spec.section.resources" // 서버에 보낼 데이터에서의 path (필수)
            items={resources} // [{title: '', value: ''}] (필수)
            inline={false} // inline속성 먹일거면 true, 아니면 빼면 됨 (선택)
          />
        </Section>
      </Section>
      <Section id="dropdown" label="Dropdown">
        <Dropdown
          name="dropdown1"
          className="btn-group"
          items={dropdownUnits} // (필수)
          required={true}
          buttonClassName="dropdown-btn" // 선택된 아이템 보여주는 button (title) 부분 className
          itemClassName="dropdown-item" // 드롭다운 아이템 리스트 전체의 className - 각 row를 의미하는 것은 아님
        />
        <ContainerDropdown
          name="containerDropdown1"
          containers={containers} // (필수)
          initContainers={initContainers}
        />
      </Section>
      <Section id="resourcedropdown" label="Resource Dropdown">
        <ResourceDropdown
          name="RD-single"
          placeholder="select one deployment"
          resources={[
            {
              kind: 'Deployment',
              namespace: 'catalog', // 옵션
              prop: 'deployment',
            },
          ]}
          type="single"
          useHookForm
        />
        <ResourceDropdown
          name="RD-multiple"
          resources={[
            {
              kind: 'Pod',
              namespace: 'default', // 옵션
              prop: 'pod',
            },
          ]}
          defaultValue={['new-pipeline-test-shtehy-sum-params-qkg57-pod-l47lv']} // 옵션
          type="multiple"
          showAll={true}
          useHookForm
        />
      </Section>
      <Section id="resourcelistdropdown" label="Resource List Dropdown">
        <ResourceListDropdown
          resourceList={ClusterResourceList} // 필수
          autocompletePlaceholder="search by name"
          placeholder="Resource Dropdown" // *single에서만 사용 가능
          type="single" // 필수 type: single / multiple
        />
        <ResourceListDropdown
          name="RLD-multiple"
          resourceList={ClusterResourceList} // 필수
          onChange={updateSelectedClusterItems} // '아이템' 선택될 때마다 호출됨
          resourceType="Cluster and Cluster Claim"
          autocompletePlaceholder="search by name"
          type="multiple" // 필수 type: single / multiple
          useHookForm
        />
        <ResourceListDropdownWithDataToolbar // react hook form 사용하지 않는 예시
          resourceList={ClusterResourceList} // 필수
          showAll={true} // 드롭다운에 all resource 라는 항목이 생긴다.
          resourceType="Cluster and Cluster Claim" // title, placeholder, all resources, chip group 에 적용되는 문구 (title, placeholder는 직접 지정하는 것의 우선순위가 더 높음)
          autocompletePlaceholder="search by name" // 검색란 placeholder
          onSelectedItemChange={onSelectedItemChange} // 선택된 아이템 '리스트' 변동될 때마다 호출되는 함수
        />
        <ResourceListDropdownWithDataToolbar // react hook form 사용하는 예시
          name="ResourceListDropdownWithDataToolbar1"
          resourceList={ClusterResourceList} // 필수
          showAll={false}
          title="select Resources" // 드롭다운 title 지정
          resourceType="Cluster and Cluster Claim"
          useHookForm
        />
      </Section>
      <Section id="numberspinner" label="Number Spinner">
        <NumberSpinner
          initialValue={0}
          min={-5}
          max={5}
          name="spinner1" // 한 페이지에 spinner 여러 개 만들 경우 name에 unique한 값을 넣어줘야 됨 (한개만 만들 땐 name이 필수 아님)
        />
      </Section>
      <Section id="tagslabel" label="Tags Label">
        <TagsLabel
          name="metadata.tags" // 서버에 보낼 데이터에서의 path (필수)
          placeholder="Enter tag" // tag가 없을 때 보여줄 placeholder (선택)
        />
      </Section>
      <Section id="list" label="Key Value List">
        <KeyValueListEditor
          name="metadata.keyvaluelist" // 서버에 보낼 데이터에서의 path (필수)
          disableReorder={false} // 순서바꾸기 제공여부 설정. 기본값은 false (선택)
        />
      </Section>
      <Section id="listviewsection1" label="Default Key/Value List View">
        <ListView name="metadata.keyValueList" addButtonText="Add Key/Value" />
      </Section>
      <Section id="listviewsection2" label="Customized List View">
        <ListView
          name="numberspinnerListView"
          addButtonText="Add Name/Num"
          headerFragment={listHeaderFragment}
          itemRenderer={listItemRenderer}
          defaultItem={{ name: '', number: 0 }}
          methods={methods}
          defaultValues={[
            {
              name: 'Item1',
              number: 3,
            },
            {
              name: 'Item2',
              number: 5,
            },
          ]}
        />
      </Section>
    </div>
  );
};

export const CreateSample: React.FC<CreateSampleProps> = props => {
  const formComponent = sampleFormFactory(props.match.params);
  const SampleFormComponent = formComponent;
  return <SampleFormComponent fixed={{}} explanation="" titleVerb="Create" onSubmitCallback={onSubmitCallback} isCreate={true} />;
};

export const onSubmitCallback = data => {
  // submit하기 전에 data를 가공해야 할 경우
  let labels = SelectorInput.objectify(data.metadata.labels);
  delete data.metadata.labels;
  data = _.defaultsDeep(data, { metadata: { labels: labels } });
  return data;
};

type CreateSampleProps = {
  match: RMatch<{
    type?: string;
  }>;
  fixed: object;
  explanation: string;
  titleVerb: string;
  saveButtonText?: string;
  isCreate: boolean;
};

type SampleFormProps = {
  onChange: Function;
  stringData: {
    [key: string]: string;
  };
  isCreate: boolean;
};
